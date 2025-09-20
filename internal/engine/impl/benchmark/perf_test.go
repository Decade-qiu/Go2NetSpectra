package test

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/impl/exact"
	"Go2NetSpectra/internal/engine/impl/sketch"
	"Go2NetSpectra/internal/model"
	"Go2NetSpectra/pkg/pcap"
	"fmt"
	"log"
	"net"
	"sync"
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

	// b.Run("CM_Parallel", run_cm_parallel)
	b.Run("SS_Parallel", run_SS_parallel)
	b.Run("SS_Exact_Parallel", run_ssexact_parallel)
	// b.Run("Exact_Parallel", run_exact_parallel)

	// run_cm(b)
	// run_ss(b)
	// run_exact(b)
}

func run_SS_parallel(b *testing.B) {
	cfg := config.SketchTaskDef{
		Name:            "SuperSpread",
		SktType:         1,
		FlowFields:      []string{"SrcIP"},
		ElementFields:   []string{"DstIP"},
		Width:           1 << 13,
		Depth:           2,
		SizeThereshold:  0,
		CountThereshold: 512,
		M: 128,
		Base: 0.5,
		Size: 5,
		B: 1.08,
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

func run_cm_parallel(b *testing.B) {
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

func run_ssexact_parallel(b *testing.B) {
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

func run_cm(b *testing.B) {
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

func run_ss(b *testing.B) {
	cfg := config.SketchTaskDef{
		Name:            "SuperSpread",
		SktType:         1,
		FlowFields:      []string{"SrcIP"},
		ElementFields:   []string{"DstIP"},
		Width:           1 << 13,
		Depth:           2,
		SizeThereshold:  0,
		CountThereshold: 512,
		M: 128,
		Base: 0.5,
		Size: 5,
		B: 1.08,
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