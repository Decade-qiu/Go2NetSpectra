package probe

import (
	"Go2NetSpectra/internal/config"
	"log"
	"net"

	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/model"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

// PacketHandler is a function that processes a received PacketInfo.
type PacketHandler func(info model.PacketInfo)

// Subscriber is responsible for subscribing to a NATS subject and processing messages.
type Subscriber struct {
	nc      *nats.Conn
	sub     *nats.Subscription
	subject string
}

// NewSubscriber creates a new NATS subscriber.
func NewSubscriber(cfg config.ProbeConfig) (*Subscriber, error) {
	nc, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		return nil, err
	}
	log.Printf("Connected to NATS server at %s", cfg.NATSURL)
	return &Subscriber{nc: nc, subject: cfg.Subject}, nil
}

// Start subscribes to the given subject and starts processing messages with the provided handler.
func (s *Subscriber) Start(handler PacketHandler) error {
	sub, err := s.nc.Subscribe(s.subject, func(msg *nats.Msg) {
		// Decode the protobuf message
		var pbPacket v1.PacketInfo
		if err := proto.Unmarshal(msg.Data, &pbPacket); err != nil {
			log.Printf("Error unmarshalling protobuf: %v", err)
			return
		}

		// Convert from protobuf type to internal model type
		info := model.PacketInfo{
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
		handler(info)
	})
	if err != nil {
		return err
	}
	s.sub = sub
	log.Printf("Subscribed to '%s'. Waiting for messages...", s.subject)
	return nil
}

// Close unsubscribes and closes the NATS connection.
func (s *Subscriber) Close() {
	if s.sub != nil {
		s.sub.Unsubscribe()
	}
	if s.nc != nil {
		s.nc.Close()
		log.Println("NATS connection closed.")
	}
}