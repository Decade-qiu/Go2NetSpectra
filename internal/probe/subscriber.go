package probe

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/model"
	"log"

	"github.com/nats-io/nats.go"
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
		info, err := UnmarshalPacketInfo(msg.Data)
		if err != nil {
			log.Printf("Error decoding thrift packet: %v", err)
			return
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
