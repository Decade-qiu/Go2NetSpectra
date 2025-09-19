package test

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/impl/exact"
	"Go2NetSpectra/internal/engine/impl/sketch"
	"Go2NetSpectra/internal/model"
	"Go2NetSpectra/pkg/pcap"
	"log"
	"net"
	"testing"
)

var packets []*model.PacketInfo

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

	b.Run("Sketch_Parallel", run_sketch_parallel)
	// b.Run("Exact_Parallel", run_exact_parallel)

	// run_sketch(b)
	// run_exact(b)
}

func run_sketch_parallel(b *testing.B) {
	cfg := config.SketchTaskDef{
		Name:            "per_src_flow",
		SktType:         0,
		FlowFields:      []string{"SrcIP"},
		ElementFields:   []string{"DstIP", "SrcPort", "DstPort", "Protocol"},
		Width:           1 << 13,
		Depth:           2,
		SizeThereshold:  4096 * 1024,
		CountThereshold: 4096,
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

func run_exact_parallel(b *testing.B) {
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

func run_sketch(b *testing.B) {
	cfg := config.SketchTaskDef{
		Name:            "per_src_flow",
		SktType:         0,
		FlowFields:      []string{"SrcIP"},
		ElementFields:   []string{"DstIP", "SrcPort", "DstPort", "Protocol"},
		Width:           1 << 13,
		Depth:           2,
		SizeThereshold:  4096 * 1024,
		CountThereshold: 4096,
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

func run_exact(b *testing.B) {
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