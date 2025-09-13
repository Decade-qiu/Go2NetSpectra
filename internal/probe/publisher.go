package probe

import (
	"Go2NetSpectra/internal/config"
	"log"

	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/model"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Publisher is responsible for publishing packet data to a NATS topic.
type Publisher struct {
	nc      *nats.Conn
	subject string
}

// NewPublisher creates a new NATS publisher.
func NewPublisher(cfg config.ProbeConfig) (*Publisher, error) {
	nc, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		return nil, err
	}
	log.Printf("Connected to NATS server at %s", cfg.NATSURL)
	return &Publisher{nc: nc, subject: cfg.Subject}, nil
}

// Publish serializes a PacketInfo to Protobuf and publishes it to the configured NATS subject.
func (p *Publisher) Publish(packetInfo *model.PacketInfo) error {
	// Convert model.PacketInfo to a protobuf message
	pbPacket := &v1.PacketInfo{
		Timestamp: timestamppb.New(packetInfo.Timestamp),
		FiveTuple: &v1.FiveTuple{
			SrcIp:    []byte(packetInfo.FiveTuple.SrcIP),
			DstIp:    []byte(packetInfo.FiveTuple.DstIP),
			SrcPort:  uint32(packetInfo.FiveTuple.SrcPort),
			DstPort:  uint32(packetInfo.FiveTuple.DstPort),
			Protocol: uint32(packetInfo.FiveTuple.Protocol),
		},
		Length: uint64(packetInfo.Length),
	}

	// Serialize to binary format
	data, err := proto.Marshal(pbPacket)
	if err != nil {
		return err
	}

	// Publish the data
	return p.nc.Publish(p.subject, data)
}

// Close drains and closes the NATS connection.
func (p *Publisher) Close() {
	if p.nc != nil {
		p.nc.Drain()
		log.Println("NATS connection drained and closed.")
	}
}