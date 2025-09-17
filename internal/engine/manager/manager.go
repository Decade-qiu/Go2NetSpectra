package manager

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"
	_ "Go2NetSpectra/internal/engine/impl/exact" // Registers exact task aggregator
	"Go2NetSpectra/internal/factory"
	"Go2NetSpectra/internal/model"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Manager orchestrates a set of aggregation tasks and their writers.
type Manager struct {
	tasks   []model.Task
	writers []model.Writer

	// Worker pool for concurrent packet processing
	packetChannel chan *v1.PacketInfo
	numWorkers    int
	workerWg      sync.WaitGroup

	// Snapshotting and Resetting resources
	period        time.Duration // Global measurement period
	done  		  chan struct{}
	snapshotterWg sync.WaitGroup
	resetterWg    sync.WaitGroup // New WaitGroup for the resetter
}

// New creates a new Manager.
func NewManager(cfg *config.Config) (*Manager, error) {
	tasks, writers, err := factory.Create(cfg)
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

	return &Manager{
		tasks:         tasks,
		writers:       writers,
		period:        period,
		done:  		   make(chan struct{}),
		packetChannel: make(chan *v1.PacketInfo, cfg.Aggregator.SizeOfPacketChannel),
		numWorkers:    cfg.Aggregator.NumWorkers,
	}, nil
}

// Start begins the manager's packet processing workers, snapshotter, and resetter goroutines.
func (m *Manager) Start() {
	// Start a dedicated snapshotter for each writer
	for _, writer := range m.writers {
		m.snapshotterWg.Add(1)
		go m.runSnapshotter(writer)
		log.Printf("Started snapshotter for a writer with interval %s", writer.GetInterval())
	}

	// Start the global resetter
	m.resetterWg.Add(1)
	go m.runResetter()
	log.Printf("Started global resetter with period %s", m.period)

	// Start the packet processing worker pool
	m.workerWg.Add(m.numWorkers)
	for i := 0; i < m.numWorkers; i++ {
		go m.worker()
	}
	log.Printf("Manager started with %d workers.", m.numWorkers)
}

// runSnapshotter runs a dedicated snapshot loop for a single writer.
func (m *Manager) runSnapshotter(writer model.Writer) {
	defer m.snapshotterWg.Done()
	interval := writer.GetInterval()
	if interval <= 0 {
		log.Printf("Invalid interval %s for writer, snapshotter will not run.", interval)
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.takeSnapshotForWriter(writer)
		case <-m.done:
			m.takeSnapshotForWriter(writer)
			return
		}
	}
}

// takeSnapshotForWriter orchestrates taking and writing a snapshot for a specific writer.
// Warning: 
// In this implementation, different tasks may complete their snapshotting at different times.
// This means that the snapshot data written for a given timestamp may not represent
// a perfectly synchronized view across all tasks. If strict synchronization is required,
// additional coordination would be needed.
func (m *Manager) takeSnapshotForWriter(writer model.Writer) {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	// Note: This concurrent snapshotting assumes that task.Snapshot() is thread-safe
	// with respect to other snapshot calls. The current implementation is safe because
	// it atomically swaps maps.
	log.Printf("Taking snapshot for writer at %s for %d tasks.", timestamp, len(m.tasks))

	var wg sync.WaitGroup
	wg.Add(len(m.tasks)) // Wait for all tasks to finish snapshotting

	for _, task := range m.tasks {
		go func(t model.Task) {
			defer wg.Done()
			snapshotData := task.Snapshot()
			if err := writer.Write(snapshotData, timestamp); err != nil {
				log.Printf("Error writing snapshot for task %s: %v", task.Name(), err)
			}
		}(task)
	}

	wg.Wait() // Wait for all tasks to complete

	timestamp = time.Now().Format("2006-01-02_15-04-05")
	log.Printf("Completed snapshot for writer at %s.", timestamp)
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

// resetAllTasks iterates through all tasks and calls their Reset method.
// Warning: 
// In this implementation, different tasks may complete their resetting at different times.
// This means that the reset operation does not happen simultaneously across all tasks.
// So in next measurement period, tasks may start from slightly different states.
// i.e., some tasks may record more packets than others.
func (m *Manager) resetAllTasks() {
	log.Printf("Resetting all tasks for new measurement period at %s for %d tasks.", time.Now().Format("2006-01-02_15-04-05"), len(m.tasks))

	var wg sync.WaitGroup
	wg.Add(len(m.tasks)) // Wait for all tasks to finish resetting

	for _, task := range m.tasks {
		go func(t model.Task) {
			defer wg.Done()
			task.Reset()
		}(task)
	}

	wg.Wait() // Wait for all tasks to complete

	log.Println("All tasks have been reset at ", time.Now().Format("2006-01-02_15-04-05"))
}

// Stop gracefully shuts down the manager.
func (m *Manager) Stop() {
	log.Println("Manager stopping...")
	// 1. Stop accepting new packets.
	close(m.packetChannel)

	// 2. Wait for all workers to finish processing buffered packets.
	log.Println("Waiting for workers to finish...")
	m.workerWg.Wait()

	// 3. Signal snapshotters and resetter to take final actions and exit.
	close(m.done)
	log.Println("Waiting for snapshotters and resetter to finish...")

	// 4. Wait for all goroutines to complete.
	m.snapshotterWg.Wait()
	m.resetterWg.Wait()
	log.Println("Manager stopped.")
}

func (m *Manager) worker() {
	defer m.workerWg.Done()
	for pbPacket := range m.packetChannel {
		info := &model.PacketInfo{
			Timestamp: pbPacket.Timestamp.AsTime(),
			Length:    int(pbPacket.Length),
			FiveTuple: model.FiveTuple{
				SrcIP:    net.IP(pbPacket.FiveTuple.SrcIp).To16(),
				DstIP:    net.IP(pbPacket.FiveTuple.DstIp).To16(),
				SrcPort:  uint16(pbPacket.FiveTuple.SrcPort),
				DstPort:  uint16(pbPacket.FiveTuple.DstPort),
				Protocol: uint8(pbPacket.FiveTuple.Protocol),
			},
		}
		for _, task := range m.tasks {
			task.ProcessPacket(info)
		}
	}
}

func (m *Manager) InputChannel() chan<- *v1.PacketInfo {
	return m.packetChannel
}