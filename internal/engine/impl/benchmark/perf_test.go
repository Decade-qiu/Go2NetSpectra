package benchmark

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/impl/exact"
	"Go2NetSpectra/internal/engine/impl/sketch"
	"Go2NetSpectra/internal/model"
	"Go2NetSpectra/internal/probe"
	"Go2NetSpectra/internal/protocol"
	"Go2NetSpectra/pkg/pcap"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"testing"

	"github.com/google/gopacket"
	gopcap "github.com/google/gopacket/pcap"
)

var packets []*model.PacketInfo
var (
	benchPacketOnce sync.Once
	benchPacketInfo *model.PacketInfo
	benchRawPacket  gopacket.Packet
	benchPacketErr  error
)

func loadBenchmarkPacket(b *testing.B) (*model.PacketInfo, gopacket.Packet) {
	b.Helper()

	benchPacketOnce.Do(func() {
		handle, err := gopcap.OpenOffline("../../../../test/data/test.pcap")
		if err != nil {
			benchPacketErr = fmt.Errorf("failed to open benchmark fixture: %w", err)
			return
		}
		defer handle.Close()

		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		packet, ok := <-packetSource.Packets()
		if !ok {
			benchPacketErr = fmt.Errorf("benchmark fixture did not yield a packet")
			return
		}

		benchRawPacket = packet
		benchPacketInfo, benchPacketErr = protocol.ParsePacket(packet)
	})

	if benchPacketErr != nil {
		b.Fatal(benchPacketErr)
	}

	return benchPacketInfo, benchRawPacket
}

func muteBenchmarkLogs() func() {
	originalWriter := log.Writer()
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(originalWriter)
	}
}

func BenchmarkProtocolParsePacketInto(b *testing.B) {
	_, rawPacket := loadBenchmarkPacket(b)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var info model.PacketInfo
		if err := protocol.ParsePacketInto(rawPacket, &info); err != nil {
			b.Fatalf("ParsePacketInto() unexpected error: %v", err)
		}
	}
}

func BenchmarkPacketCodecRoundTrip(b *testing.B) {
	packetInfo, _ := loadBenchmarkPacket(b)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data, err := probe.MarshalPacketInfo(nil, packetInfo)
		if err != nil {
			b.Fatalf("MarshalPacketInfo() unexpected error: %v", err)
		}
		if _, err := probe.UnmarshalPacketInfo(data); err != nil {
			b.Fatalf("UnmarshalPacketInfo() unexpected error: %v", err)
		}
	}
}

func BenchmarkExactTaskProcessPacket(b *testing.B) {
	packetInfo, _ := loadBenchmarkPacket(b)
	restoreLogs := muteBenchmarkLogs()
	defer restoreLogs()
	task := exact.New("exact-srcip", []string{"SrcIP"}, 64)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		task.ProcessPacket(packetInfo)
		if i%1024 == 1023 {
			task.Reset()
		}
	}
}

func BenchmarkCountMinTaskProcessPacket(b *testing.B) {
	packetInfo, _ := loadBenchmarkPacket(b)
	restoreLogs := muteBenchmarkLogs()
	defer restoreLogs()
	task := sketch.New(config.SketchTaskDef{
		Name:           "bench-count-min",
		SketchType:     0,
		FlowFields:     []string{"SrcIP"},
		ElementFields:  []string{"DstIP", "SrcPort", "DstPort", "Protocol"},
		Width:          1 << 10,
		Depth:          2,
		SizeThreshold:  1,
		CountThreshold: 1,
	})
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		task.ProcessPacket(packetInfo)
		if i%1024 == 1023 {
			task.Reset()
		}
	}
}

func BenchmarkSuperSpreadTaskProcessPacket(b *testing.B) {
	packetInfo, _ := loadBenchmarkPacket(b)
	restoreLogs := muteBenchmarkLogs()
	defer restoreLogs()
	task := sketch.New(config.SketchTaskDef{
		Name:           "bench-super-spread",
		SketchType:     1,
		FlowFields:     []string{"DstIP"},
		ElementFields:  []string{"SrcIP"},
		Width:          1 << 10,
		Depth:          2,
		CountThreshold: 1,
		M:              32,
		Size:           5,
		Base:           0.5,
		B:              1.08,
	})
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		task.ProcessPacket(packetInfo)
		if i%1024 == 1023 {
			task.Reset()
		}
	}
}

func BenchmarkAggregator(b *testing.B) {
	pcapFilePath := "../../../../test/data/caida.pcap"
	pcapReader, err := pcap.NewReader(pcapFilePath)
	if err != nil {
		b.Fatalf("Failed to open pcap file: %v", err)
	}
	defer pcapReader.Close()
	log.Printf("Reading packets from '%s'...", pcapFilePath)

	packetChannel := make(chan *v1.PacketInfo, 10000)
	go func() {
		pcapReader.ReadPackets(packetChannel)
		close(packetChannel)
	}()

	for pbPacket := range packetChannel {
		info := &model.PacketInfo{
			Timestamp: pbPacket.Timestamp.AsTime(),
			Length:    int(pbPacket.Length),
			FiveTuple: model.FiveTuple{
				SrcIP:    net.IP(pbPacket.FiveTuple.SrcIp).To16(),
				DstIP:    net.IP(pbPacket.FiveTuple.DstIp).To16(),
				SrcPort:  uint16(pbPacket.FiveTuple.SrcPort),
				DstPort:  uint16(pbPacket.FiveTuple.DstPort),
				Protocol: uint8(pbPacket.FiveTuple.Protocol),
			},
		}
		packets = append(packets, info)
	}

	// b.Run("CM_Parallel", run_cm_parallel)
	b.Run("SS_Parallel", runSSParallel)
	b.Run("SS_Exact_Parallel", runSSExactParallel)
	// b.Run("Exact_Parallel", run_exact_parallel)

	// run_cm(b)
	// run_ss(b)
	// run_exact(b)
}

func runSSParallel(b *testing.B) {
	cfg := config.SketchTaskDef{
		Name:           "SuperSpread",
		SketchType:     1,
		FlowFields:     []string{"SrcIP"},
		ElementFields:  []string{"DstIP"},
		Width:          1 << 13,
		Depth:          2,
		SizeThreshold:  0,
		CountThreshold: 512,
		M:              128,
		Base:           0.5,
		Size:           5,
		B:              1.08,
	}

	task := sketch.New(cfg)

	b.Run("Insert_SS_Parallel", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for _, pkt := range packets {
					task.ProcessPacket(pkt)
				}
			}
		})
	})

	b.Run("Query_SS_Parallel", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for _, pkt := range packets {
					task.Query(pkt.FiveTuple.SrcIP.To16())
				}
			}
		})
	})
}

func runCmParallel(b *testing.B) {
	cfg := config.SketchTaskDef{
		Name:           "per_src_flow",
		SketchType:     0,
		FlowFields:     []string{"SrcIP"},
		ElementFields:  []string{"DstIP", "SrcPort", "DstPort", "Protocol"},
		Width:          1 << 13,
		Depth:          2,
		SizeThreshold:  4096 * 1024,
		CountThreshold: 4096,
	}

	task := sketch.New(cfg)

	b.Run("Insert_Sketch_Parallel", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for _, pkt := range packets {
					task.ProcessPacket(pkt)
				}
			}
		})
	})

	b.Run("Query_Sketch_Parallel", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for _, pkt := range packets {
					task.Query(pkt.FiveTuple.SrcIP.To16())
				}
			}
		})
	})
}

func runExactParallel(b *testing.B) {
	task := exact.New("exact_per_src", []string{"SrcIP"}, 64)

	b.Run("Insert_Exact_Parallel", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for _, pkt := range packets {
					task.ProcessPacket(pkt)
				}
			}
		})
	})

	b.Run("Query_Exact_Parallel", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for _, pkt := range packets {
					task.Query(pkt.FiveTuple.SrcIP.To16())
				}
			}
		})
	})
}

func runSSExactParallel(b *testing.B) {
	spreadMap := make(map[string]map[string]bool)
	var mu sync.Mutex

	b.Run("Insert_spreadmap_Parallel", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for _, pkt := range packets {
					key := pkt.FiveTuple.SrcIP.String()
					elem := fmt.Sprintf("%s",
						pkt.FiveTuple.DstIP.String())
					mu.Lock()
					if _, exists := spreadMap[key]; !exists {
						spreadMap[key] = make(map[string]bool)
					}
					spreadMap[key][elem] = true
					mu.Unlock()
				}
			}
		})
	})

	b.Run("Query_spreadmap_Parallel", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for _, pkt := range packets {
					_ = len(spreadMap[pkt.FiveTuple.SrcIP.String()])
				}
			}
		})
	})
}

func runCm(b *testing.B) {
	cfg := config.SketchTaskDef{
		Name:           "per_src_flow",
		SketchType:     0,
		FlowFields:     []string{"SrcIP"},
		ElementFields:  []string{"DstIP", "SrcPort", "DstPort", "Protocol"},
		Width:          1 << 13,
		Depth:          2,
		SizeThreshold:  4096 * 1024,
		CountThreshold: 4096,
	}

	task := sketch.New(cfg)

	b.Run("Insert_Sketch", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, pkt := range packets {
				task.ProcessPacket(pkt)
			}
		}
	})

	b.Run("Query_Sketch", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, pkt := range packets {
				task.Query(pkt.FiveTuple.SrcIP.To16())
			}
		}
	})
}

func runSs(b *testing.B) {
	cfg := config.SketchTaskDef{
		Name:           "SuperSpread",
		SketchType:     1,
		FlowFields:     []string{"SrcIP"},
		ElementFields:  []string{"DstIP"},
		Width:          1 << 13,
		Depth:          2,
		SizeThreshold:  0,
		CountThreshold: 512,
		M:              128,
		Base:           0.5,
		Size:           5,
		B:              1.08,
	}

	task := sketch.New(cfg)

	b.Run("Insert_SS", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, pkt := range packets {
				task.ProcessPacket(pkt)
			}
		}
	})

	b.Run("Query_SS", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, pkt := range packets {
				task.Query(pkt.FiveTuple.SrcIP.To16())
			}
		}
	})
}

func runExact(b *testing.B) {
	task := exact.New("exact_per_src", []string{"SrcIP"}, 64)

	b.Run("Insert_Exact", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, pkt := range packets {
				task.ProcessPacket(pkt)
			}
		}
	})

	b.Run("Query_Exact", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, pkt := range packets {
				task.Query(pkt.FiveTuple.SrcIP.To16())
			}
		}
	})
}
