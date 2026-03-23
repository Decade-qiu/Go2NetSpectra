package manager

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/alerter"
	"Go2NetSpectra/internal/config"
	_ "Go2NetSpectra/internal/engine/impl/exact"  // Registers exact task aggregator
	_ "Go2NetSpectra/internal/engine/impl/sketch" // Registers sketch task aggregator
	"Go2NetSpectra/internal/factory"
	"Go2NetSpectra/internal/model"
	"Go2NetSpectra/internal/notification"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Manager orchestrates a set of aggregation tasks and their writers.
type Manager struct {
	taskGroups []factory.TaskGroup
	alerter    *alerter.Alerter

	// Worker pool for concurrent packet processing
	packetChannel chan *v1.PacketInfo
	numWorkers    int
	workerWg      sync.WaitGroup

	// Snapshotting and Resetting resources
	period        time.Duration // Global measurement period
	done          chan struct{}
	stopOnce      sync.Once
	snapshotterWg sync.WaitGroup
	resetterWg    sync.WaitGroup // New WaitGroup for the resetter
}

// NewManager creates a new Manager.
func NewManager(cfg *config.Config) (*Manager, error) {
	taskGroups, err := factory.Create(cfg)
	if err != nil {
		return nil, err
	}

	period, err := time.ParseDuration(cfg.Aggregator.Period)
	if err != nil {
		return nil, fmt.Errorf("invalid aggregator period: %w", err)
	}
	if period <= 0 {
		return nil, fmt.Errorf("aggregator period must be a positive duration")
	}

	var alertr *alerter.Alerter
	if cfg.Alerter.Enabled {
		var allTasks []model.Task
		for _, group := range taskGroups {
			allTasks = append(allTasks, group.Tasks...)
		}

		// For now, we only initialize the email notifier. This can be expanded later.
		var notifier model.Notifier
		if cfg.SMTP.Host != "" { // Simple check to see if email is configured
			notifier = notification.NewEmailNotifier(cfg.SMTP)
		}

		if notifier != nil {
			var err error
			alertr, err = alerter.NewAlerter(&cfg.Alerter, allTasks, notifier)
			if err != nil {
				return nil, fmt.Errorf("failed to create alerter: %w", err)
			}
			log.Println("Alerter enabled and initialized.")
		} else {
			log.Println("Alerter is enabled in config, but no notifiers are configured. Alerter will not run.")
		}
	}

	return &Manager{
		taskGroups:    taskGroups,
		alerter:       alertr,
		period:        period,
		done:          make(chan struct{}),
		packetChannel: make(chan *v1.PacketInfo, cfg.Aggregator.SizeOfPacketChannel),
		numWorkers:    max(1, cfg.Aggregator.NumWorkers),
	}, nil
}

// Start begins the manager's packet processing workers, snapshotter, and resetter goroutines.
func (m *Manager) Start() {
	// For each group, start a dedicated snapshotter for each of its writers.
	for _, group := range m.taskGroups {
		for _, writer := range group.Writers {
			m.snapshotterWg.Add(1)
			// Pass the group-specific tasks to the snapshotter
			go m.runSnapshotter(writer, group.Tasks)
			log.Printf("Started snapshotter for a writer with interval %s, handling %d tasks.", writer.Interval(), len(group.Tasks))
		}
	}

	// Start the global resetter for all tasks across all groups.
	m.resetterWg.Add(1)
	go m.runResetter()
	log.Printf("Started global resetter with period %s", m.period)

	// Start the independent alerter goroutine if it's enabled.
	if m.alerter != nil {
		m.alerter.Start()
	}

	// Start the packet processing worker pool.
	m.workerWg.Add(m.numWorkers)
	for i := 0; i < m.numWorkers; i++ {
		go m.worker()
	}
	log.Printf("Manager started with %d workers.", m.numWorkers)
}

// runSnapshotter runs a dedicated snapshot loop for a single writer and its associated tasks.
func (m *Manager) runSnapshotter(writer model.Writer, tasks []model.Task) {
	defer m.snapshotterWg.Done()
	interval := writer.Interval()
	if interval <= 0 {
		log.Printf("Invalid interval %s for writer, snapshotter will not run.", interval)
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.takeSnapshotForWriter(writer, tasks)
		case <-m.done:
			m.takeSnapshotForWriter(writer, tasks)
			return
		}
	}
}

// takeSnapshotForWriter orchestrates taking and writing a snapshot for a specific writer.
func (m *Manager) takeSnapshotForWriter(writer model.Writer, tasks []model.Task) {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	log.Printf("Taking snapshot for writer at %s for %d tasks.", timestamp, len(tasks))

	var wg sync.WaitGroup
	wg.Add(len(tasks)) // Wait for all tasks in this group to finish snapshotting

	for _, task := range tasks {
		go func(t model.Task) {
			defer wg.Done()
			snapshotData := t.Snapshot()
			if err := writer.Write(snapshotData, timestamp, t.Name(), t.Fields(), t.DecodeFlowFunc()); err != nil {
				log.Printf("Error writing snapshot for task %s: %v", t.Name(), err)
			}
		}(task)
	}

	wg.Wait() // Wait for all tasks in this group to complete

	log.Printf("Completed snapshot for writer at %s.", time.Now().Format("2006-01-02_15-04-05"))
}

// runResetter runs a dedicated loop to reset all tasks periodically.
func (m *Manager) runResetter() {
	defer m.resetterWg.Done()
	ticker := time.NewTicker(m.period)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.resetAllTasks()
		case <-m.done:
			log.Println("Resetter shutting down.")
			return
		}
	}
}

// resetAllTasks iterates through all tasks across all groups and calls their Reset method.
func (m *Manager) resetAllTasks() {
	log.Printf("Resetting all tasks for new measurement period at %s", time.Now().Format("2006-01-02_15-04-05"))
	var wg sync.WaitGroup
	for _, group := range m.taskGroups {
		wg.Add(len(group.Tasks))
		for _, task := range group.Tasks {
			go func(t model.Task) {
				defer wg.Done()
				t.Reset()
			}(task)
		}
	}
	wg.Wait() // Wait for all tasks to complete
	log.Printf("All tasks have been reset at %s", time.Now().Format("2006-01-02_15-04-05"))
}

// Stop gracefully shuts down the manager.
func (m *Manager) Stop() {
	m.stopOnce.Do(func() {
		log.Println("Manager stopping...")
		close(m.packetChannel)

		log.Println("Waiting for workers to finish...")
		m.workerWg.Wait()

		close(m.done)
		log.Println("Waiting for snapshotters and resetter to finish...")

		m.snapshotterWg.Wait()
		m.resetterWg.Wait()

		if m.alerter != nil {
			m.alerter.Stop()
		}

		log.Println("Manager stopped.")
	})
}

func (m *Manager) worker() {
	defer m.workerWg.Done()
	for pbPacket := range m.packetChannel {
		if err := m.processProtoPacket(pbPacket); err != nil {
			log.Printf("Manager failed to process protobuf packet: %v", err)
		}
	}
}

// InputChannel returns the protobuf packet input channel consumed by the worker pool.
func (m *Manager) InputChannel() chan<- *v1.PacketInfo {
	return m.packetChannel
}

func (m *Manager) processProtoPacket(pbPacket *v1.PacketInfo) error {
	if pbPacket == nil {
		return fmt.Errorf("nil protobuf packet")
	}
	if pbPacket.FiveTuple == nil {
		return fmt.Errorf("nil protobuf five tuple")
	}

	info := &model.PacketInfo{
		Timestamp: pbPacket.Timestamp.AsTime(),
		Length:    int(pbPacket.Length),
		FiveTuple: model.FiveTuple{
			SrcIP:    append(net.IP(nil), pbPacket.FiveTuple.SrcIp...),
			DstIP:    append(net.IP(nil), pbPacket.FiveTuple.DstIp...),
			SrcPort:  uint16(pbPacket.FiveTuple.SrcPort),
			DstPort:  uint16(pbPacket.FiveTuple.DstPort),
			Protocol: uint8(pbPacket.FiveTuple.Protocol),
		},
	}

	for _, group := range m.taskGroups {
		for _, task := range group.Tasks {
			task.ProcessPacket(info)
		}
	}

	return nil
}
