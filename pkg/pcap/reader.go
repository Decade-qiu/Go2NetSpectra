package pcap

import (
	"Go2NetSpectra/internal/core/model"
	"Go2NetSpectra/internal/engine/protocol"

	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
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

// ReadPackets reads all packets from the pcap file, parses them concurrently,
// and sends the parsed PacketInfo to the provided channel.
func (r *Reader) ReadPackets(out chan<- *model.PacketInfo) {
	var wg sync.WaitGroup

	packetSource := gopacket.NewPacketSource(r.handle, r.handle.LinkType())
	for packet := range packetSource.Packets() {
		wg.Add(1)
		go func(packet gopacket.Packet) {
			defer wg.Done()

			info, err := protocol.ParsePacket(packet.Data())
			if err != nil {
				// In a real-time system, we might want to be more selective about logging
				// to avoid overwhelming logs, but for offline analysis this is fine.
				// log.Printf("Error parsing packet: %v", err)
			} else {
				out <- info
			}
		}(packet)
	}

	wg.Wait()
}

