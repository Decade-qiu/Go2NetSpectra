package manager

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"
	_ "Go2NetSpectra/internal/engine/impl/exact"
	"Go2NetSpectra/internal/factory"
	"Go2NetSpectra/internal/model"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Manager orchestrates a set of aggregation tasks.
// It implements the model.Manager interface.
type Manager struct {
	tasks            []model.Task
	snapshotInterval time.Duration

	// Worker pool for concurrent packet processing
	packetChannel chan *v1.PacketInfo
	numWorkers    int
	workerWg      sync.WaitGroup

	// Snapshot writing
	writer           model.Writer
	done             chan struct{}
	snapshotterWg    sync.WaitGroup
}

// New creates a new Manager.
func NewManager(cfg *config.Config) (*Manager, error) {
	snapshotInterval, err := time.ParseDuration(cfg.Aggregator.SnapshotInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid snapshot_interval: %w", err)
	}

	var tasks []model.Task
	var writer model.Writer

	// Use the factory pattern to create tasks and writers based on config type
	tasks, writer, err = factory.Create(cfg)
	if err != nil {
		return nil, err
	}

	return &Manager{
		tasks:            tasks,
		snapshotInterval: snapshotInterval,
		writer:           writer,
		done:             make(chan struct{}),
		packetChannel:    make(chan *v1.PacketInfo, cfg.Aggregator.SizeOfPacketChannel),
		numWorkers:       cfg.Aggregator.NumWorkers,
	}, nil
}

// InputChannel returns the channel to which protobuf packets should be sent.
func (m *Manager) InputChannel() chan<- *v1.PacketInfo {
	return m.packetChannel
}

// Start begins the manager's snapshotting ticker and worker pool.
func (m *Manager) Start() {
	// Start the snapshotter
	m.snapshotterWg.Add(1)
	go m.snapshotter()

	// Start the worker pool
	m.workerWg.Add(m.numWorkers)
	for i := 0; i < m.numWorkers; i++ {
		go m.worker()
	}
	log.Printf("Manager started with %d workers.", m.numWorkers)
}

// Stop gracefully shuts down the manager.
func (m *Manager) Stop() {
	log.Println("Manager stopping...")
	// 1. Stop accepting new packets.
	close(m.packetChannel)

	// 2. Wait for all workers to finish processing buffered packets.
	log.Println("Waiting for workers to finish...")
	m.workerWg.Wait()

	// 3. Signal the snapshotter to take a final snapshot and exit.
	close(m.done)

	// 4. Wait for the snapshotter to finish.
	m.snapshotterWg.Wait()
	log.Println("Manager stopped.")
}

// worker is a goroutine that consumes packets from the channel and processes them.
func (m *Manager) worker() {
	defer m.workerWg.Done()
	for pbPacket := range m.packetChannel {
		// Convert from protobuf type to internal model type
		info := &model.PacketInfo{
			Timestamp: pbPacket.Timestamp.AsTime(),
			Length:    int(pbPacket.Length),
			FiveTuple: model.FiveTuple{
				SrcIP:    net.IP(pbPacket.FiveTuple.SrcIp),
				DstIP:    net.IP(pbPacket.FiveTuple.DstIp),
				SrcPort:  uint16(pbPacket.FiveTuple.SrcPort),
				DstPort:  uint16(pbPacket.FiveTuple.DstPort),
				Protocol: uint8(pbPacket.FiveTuple.Protocol),
			},
		}
		// Feed the packet to all underlying tasks.
		// log.Println("Processing packet:", info)
		for _, task := range m.tasks {
			task.ProcessPacket(info)
		}
	}
}

// snapshotter periodically triggers a snapshot of all tasks.
func (m *Manager) snapshotter() {
	defer m.snapshotterWg.Done()
	ticker := time.NewTicker(m.snapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.takeSnapshot()
		case <-m.done:
			m.takeSnapshot()
			log.Println("Final snapshot taken at shutdown.")
			return
		}
	}
}

// takeSnapshot orchestrates the process of taking and writing a snapshot.
func (m *Manager) takeSnapshot() {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	log.Printf("Starting snapshot for timestamp %s... for %d tasks", timestamp, len(m.tasks))

	wg := sync.WaitGroup{}
	wg.Add(len(m.tasks))
	for _, task := range m.tasks {
		go func(task model.Task) {
			defer wg.Done()
			snapshotData := task.Snapshot()
			if err := m.writer.Write(snapshotData, timestamp); err != nil {
				log.Printf("Error writing snapshot for %s: %v", task.Name(), err)
			}
		}(task)
	}
	wg.Wait()
}