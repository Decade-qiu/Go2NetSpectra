package sketch

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/engine/impl/sketch/statistic"
	"Go2NetSpectra/internal/model"
	"Go2NetSpectra/pkg/pcap"
	"log"
	"net"
	"slices"
	"sync"
	"testing"
)

type kv struct {
	Key   string
	Count uint32
}

func TestCountMin(t *testing.T) {
	pcapFilePath := "../../../../test/data/caida.pcap"
	pcapReader, err := pcap.NewReader(pcapFilePath)
	if err != nil {
		log.Fatalf("Failed to open pcap file: %v", err)
	}
	defer pcapReader.Close()
	log.Printf("Reading packets from '%s'...", pcapFilePath)

	var packetChannel chan *v1.PacketInfo = make(chan *v1.PacketInfo, 1000)
	defer close(packetChannel)

	// cm
	task := New("per_src_flow", []string{"SrcIP"}, []string{"dst_ip", "src_port", "dst_port", "protocol"}, 1 << 20, 3, 1024)
	// map
	hashmap := make(map[string]uint32)

	var wg sync.WaitGroup
	wg.Add(1)

	go func () {
		defer wg.Done()
		for pbPacket := range packetChannel {
			info := &model.PacketInfo{
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

			task.ProcessPacket(info)

			key := info.FiveTuple.SrcIP.String()
			hashmap[key] += 1
		}
	}()

	log.Println("Start processing...")
	pcapReader.ReadPackets(packetChannel)
	wg.Wait()
	log.Println("Finished reading all packets from pcap file.")
	

	// validate mre
	relativeErrorSum := 0.0
	for key, actualCount := range hashmap {
		flow := net.ParseIP(key).To16()
		estimatedCount := task.Query(flow)
		relativeError := float64(estimatedCount-actualCount) / float64(actualCount)
		if relativeError < 0 {
			relativeError = -relativeError
		}
		relativeErrorSum += relativeError
	}
	avgRelativeError := relativeErrorSum / float64(len(hashmap))
	log.Printf("Average Relative Error: %.4f\n", avgRelativeError)

	// topk
	log.Println("Top-k Heavy Hitters:")
	topk := task.Snapshot().([]statistic.HeavyRecord)
	topN := 5
	for i := 0; i < topN && i < len(topk); i++ {
		log.Printf("%s : %d\n", net.IP(topk[i].Flow).String(), topk[i].Count)
	}
	
	var counts []kv
	for k, v := range hashmap {
		counts = append(counts, kv{Key: k, Count: v})
	}

	slices.SortFunc(counts, func(a, b kv) int {
		return int(b.Count) - int(a.Count)
	})

	log.Printf("Top-%d (ground truth from map):\n", topN)
	for i := 0; i < topN; i++ {
		log.Printf("%s : %d\n", counts[i].Key, counts[i].Count)
	}
}

