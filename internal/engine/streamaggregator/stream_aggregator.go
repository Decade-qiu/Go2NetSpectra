package streamaggregator

import (
	"log"

	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/manager"
	"Go2NetSpectra/internal/model"
	"Go2NetSpectra/internal/probe"

	"github.com/nats-io/nats.go"
)

// StreamAggregator consumes packets from NATS and uses a model.Manager to aggregate them.
type StreamAggregator struct {
	nc           *nats.Conn
	sub          *nats.Subscription
	manager      *manager.Manager
	inputChannel chan<- *model.PacketInfo
	natsURL      string
	natsSubject  string
}

// NewStreamAggregator creates a new real-time stream aggregator.
func NewStreamAggregator(cfg *config.Config) (*StreamAggregator, error) {
	// The new manager will handle the actual aggregation.
	mgr, err := manager.NewManager(cfg)
	if err != nil {
		return nil, err
	}

	return &StreamAggregator{
		manager:      mgr,
		inputChannel: mgr.InputChannel(), // Get the channel from the manager
		natsURL:      cfg.Probe.NATSURL,
		natsSubject:  cfg.Probe.Subject,
	}, nil
}

// Start connects to NATS, starts the underlying manager, and begins processing messages.
func (sa *StreamAggregator) Start() error {
	log.Println("StreamAggregator starting for nats: ", sa.natsURL)
	nc, err := nats.Connect(sa.natsURL)
	if err != nil {
		return err
	}
	sa.nc = nc

	// The manager starts its own worker pool and snapshotter.
	sa.manager.Start()

	sa.sub, err = sa.nc.Subscribe(sa.natsSubject, sa.handlePacket)
	if err != nil {
		sa.nc.Close()
		sa.nc = nil
		sa.manager.Stop()
		return err
	}
	log.Printf("StreamAggregator subscribed to '%s'", sa.natsSubject)
	return nil
}

// Stop gracefully shuts down the aggregator.
func (sa *StreamAggregator) Stop() {
	log.Println("StreamAggregator stopping...")
	if sa.sub != nil {
		if err := sa.sub.Unsubscribe(); err != nil {
			log.Printf("StreamAggregator failed to unsubscribe: %v", err)
		}
	}
	if sa.nc != nil {
		if err := sa.nc.Drain(); err != nil {
			log.Printf("StreamAggregator failed to drain NATS connection: %v", err)
			sa.nc.Close()
		}
	}
	// Stop the underlying manager, which will close the input channel
	// and wait for workers to finish before taking a final snapshot.
	sa.manager.Stop()
	log.Println("StreamAggregator stopped.")
}

// handlePacket decodes the message and passes it to the manager's channel.
func (sa *StreamAggregator) handlePacket(msg *nats.Msg) {
	packet, err := probe.UnmarshalPacketInfo(msg.Data)
	if err != nil {
		log.Printf("Error unmarshalling thrift packet: %v", err)
		return
	}

	// Pass the decoded packet to the manager's channel for concurrent processing.
	sa.inputChannel <- &packet
}
