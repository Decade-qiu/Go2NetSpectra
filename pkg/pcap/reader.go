package pcap

import (
	"Go2NetSpectra/internal/core/model"
	"Go2NetSpectra/internal/engine/protocol"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"log"
)

// Reader reads packets from a pcap file.
type Reader struct {
	handle *pcap.Handle
}

// NewReader creates a new pcap reader for the given file path.
func NewReader(filePath string) (*Reader, error) {
	handle, err := pcap.OpenOffline(filePath)
	if err != nil {
		return nil, err
	}
	return &Reader{handle: handle}, nil
}

// Close closes the pcap handle.
func (r *Reader) Close() {
	r.handle.Close()
}

// ReadPackets reads all packets from the pcap file and sends the parsed
// PacketInfo to the provided channel. It closes the channel when done.
func (r *Reader) ReadPackets(out chan<- *model.PacketInfo) {
	

	packetSource := gopacket.NewPacketSource(r.handle, r.handle.LinkType())
	for packet := range packetSource.Packets() {
		info, err := protocol.ParsePacket(packet.Data())
		if err != nil {
			// We log errors from the parser but continue processing.
			// This could be because of unsupported packet types or corrupt data.
			log.Printf("Error parsing packet: %v", err)
			continue
		}
		out <- info
	}
}
