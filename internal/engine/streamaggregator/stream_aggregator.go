package streamaggregator

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/manager"
	"log"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

const (
	// NATS connection details (hardcoded for now, should be moved to config)
	natsURL   = nats.DefaultURL
	natsSubject = "gons.packets.raw"
)

// StreamAggregator consumes packets from NATS and uses a model.Manager to aggregate them.
type StreamAggregator struct {
	nc           *nats.Conn
	sub          *nats.Subscription
	manager      *manager.Manager
	inputChannel chan<- *v1.PacketInfo
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
	}, nil
}

// Start connects to NATS, starts the underlying manager, and begins processing messages.
func (sa *StreamAggregator) Start() {
	log.Println("StreamAggregator starting...")
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("StreamAggregator failed to connect to NATS: %v", err)
	}
	sa.nc = nc

	// The manager starts its own worker pool and snapshotter.
	sa.manager.Start()

	sa.sub, err = sa.nc.Subscribe(natsSubject, sa.handlePacket)
	if err != nil {
		log.Fatalf("StreamAggregator failed to subscribe: %v", err)
	}
	log.Printf("StreamAggregator subscribed to '%s'", natsSubject)
}

// Stop gracefully shuts down the aggregator.
func (sa *StreamAggregator) Stop() {
	log.Println("StreamAggregator stopping...")
	if sa.sub != nil {
		sa.sub.Unsubscribe()
	}
	if sa.nc != nil {
		sa.nc.Close()
	}
	// Stop the underlying manager, which will close the input channel
	// and wait for workers to finish before taking a final snapshot.
	sa.manager.Stop()
	log.Println("StreamAggregator stopped.")
}

// handlePacket decodes the message and passes it to the manager's channel.
func (sa *StreamAggregator) handlePacket(msg *nats.Msg) {
	var pbPacket v1.PacketInfo
	if err := proto.Unmarshal(msg.Data, &pbPacket); err != nil {
		log.Printf("Error unmarshalling protobuf: %v", err)
		return
	}

	// Pass the protobuf packet to the manager's channel for concurrent processing.
	sa.inputChannel <- &pbPacket
}