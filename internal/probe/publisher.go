package probe

import (
	"log"
	"sync"

	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/model"
	"Go2NetSpectra/internal/probe/persistent"

	"github.com/google/gopacket"
	"github.com/nats-io/nats.go"
)

var publisherBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, 256)
	},
}

// Publisher is responsible for publishing packet data to a NATS topic.
type Publisher struct {
	nc                *nats.Conn
	subject           string
	persistenceWorker *persistent.Worker
}

// NewPublisher creates a new NATS publisher.
func NewPublisher(cfg config.ProbeConfig) (*Publisher, error) {
	nc, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		return nil, err
	}
	log.Printf("Connected to NATS server at %s", cfg.NATSURL)

	p := &Publisher{
		nc:      nc,
		subject: cfg.Subject,
	}

	// Initialize persistence worker if enabled
	if cfg.Persistence.Enabled {
		p.persistenceWorker, err = persistent.NewWorker(cfg.Persistence)
		if err != nil {
			log.Printf("Warning: Failed to initialize persistence worker: %v", err)
			p.persistenceWorker = nil
		}
	}

	return p, nil
}

// Publish serializes a PacketInfo to Thrift and publishes it to the configured NATS subject.
// If persistence is enabled, it also enqueues the packet for local writing.
func (p *Publisher) Publish(rawPacket gopacket.Packet, packetInfo *model.PacketInfo) error {
	// Asynchronously write to local file if persistence is enabled
	if p.persistenceWorker != nil {
		container := &persistent.PacketContainer{
			RawPacket:  rawPacket,
			PacketInfo: packetInfo,
		}
		p.persistenceWorker.Enqueue(container)
	}

	buffer := publisherBufferPool.Get().([]byte)
	data, err := MarshalPacketInfo(buffer, packetInfo)
	if err != nil {
		publisherBufferPool.Put(buffer[:0])
		return err
	}
	defer publisherBufferPool.Put(data[:0])

	return p.nc.Publish(p.subject, data)
}

// Close drains and closes the NATS connection and stops the persistence worker.
func (p *Publisher) Close() {
	if p.persistenceWorker != nil {
		p.persistenceWorker.Stop()
	}
	if p.nc != nil {
		p.nc.Drain()
		log.Println("NATS connection drained and closed.")
	}
}
