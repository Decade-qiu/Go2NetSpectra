package pcap

import (
	"Go2NetSpectra/internal/engine/protocol"
	"Go2NetSpectra/internal/model"
	"log"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

// Reader reads packets from a pcap file.
type Reader struct {
	handle *pcap.Handle
	total, failed  int
}

// NewReader creates a new pcap reader for the given file path.
func NewReader(filePath string) (*Reader, error) {
	handle, err := pcap.OpenOffline(filePath)
	if err != nil {
		return nil, err
	}
	return &Reader{handle: handle, total: 0, failed: 0}, nil
}

// Close closes the pcap handle.
func (r *Reader) Close() {
	r.handle.Close()
}

// ReadPackets reads all packets from the pcap file and sends the parsed
// PacketInfo to the provided channel.
func (r *Reader) ReadPackets(out chan<- *model.PacketInfo) {
	defer func() {
		log.Println("Total packets read:", r.total, "Failed to parse:", r.failed)
	}()
	packetSource := gopacket.NewPacketSource(r.handle, r.handle.LinkType())
	for packet := range packetSource.Packets() {
		r.total++
		info, err := protocol.ParsePacket(packet)
		if err != nil {
			// We can ignore parsing errors for now
			r.failed++
			continue
		}
		out <- info	
	}
}