package exactaggregator

import (
	"Go2NetSpectra/internal/model"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

const defaultShardCount = 256

// KeyedAggregator performs aggregation for a specific set of key fields using a sharded map
// for improved concurrency.
type KeyedAggregator struct {
	Name        string
	KeyFields   []string
	shards      []*Shard
	shardCount  uint32
}

// NewKeyedAggregator creates a new sharded aggregator.
func NewKeyedAggregator(name string, keyFields []string, NumShards int) *KeyedAggregator {
	if NumShards <= 0 || NumShards > 65536 {
		NumShards = defaultShardCount
	}
	agg := &KeyedAggregator{
		Name:        name,
		KeyFields:   keyFields,
		shards:      make([]*Shard, defaultShardCount),
		shardCount:  defaultShardCount,
	}
	for i := 0; i < int(defaultShardCount); i++ {
		agg.shards[i] = &Shard{
			Flows: make(map[string]*Flow),
		}
	}
	return agg
}

// getShard returns the appropriate shard for a given key.
func (ka *KeyedAggregator) getShard(key string) *Shard {
	hasher := fnv.New32a()
	hasher.Write([]byte(key))
	return ka.shards[hasher.Sum32()%ka.shardCount]
}

// generateKey creates a unique string key for a packet based on the aggregator's KeyFields.
func (ka *KeyedAggregator) generateKey(ft model.FiveTuple) (string, error) {
	var parts []string
	for _, field := range ka.KeyFields {
		switch field {
		case "SrcIP":
			parts = append(parts, ft.SrcIP.String())
		case "DstIP":
			parts = append(parts, ft.DstIP.String())
		case "SrcPort":
			parts = append(parts, strconv.Itoa(int(ft.SrcPort)))
		case "DstPort":
			parts = append(parts, strconv.Itoa(int(ft.DstPort)))
		case "Protocol":
			parts = append(parts, strconv.Itoa(int(ft.Protocol)))
		default:
			return "", fmt.Errorf("unknown key field: %s", field)
		}
	}
	return strings.Join(parts, "-"), nil
}

// ProcessPacket processes a single packet, creating or updating a flow in the correct shard.
func (ka *KeyedAggregator) ProcessPacket(packetInfo *model.PacketInfo) {
	key, err := ka.generateKey(packetInfo.FiveTuple)
	if err != nil {
		fmt.Printf("Error generating key for aggregator '%s': %v\n", ka.Name, err)
		return
	}

	shard := ka.getShard(key)
	shard.Mu.Lock()
	defer shard.Mu.Unlock()

	if flow, ok := shard.Flows[key]; ok {
		// Flow exists, update it
		flow.EndTime = packetInfo.Timestamp
		flow.PacketCount++
		flow.ByteCount += uint64(packetInfo.Length)
	} else {
		// Flow does not exist, create a new one
		shard.Flows[key] = &Flow{
			Key:         key,
			StartTime:   packetInfo.Timestamp,
			EndTime:     packetInfo.Timestamp,
			PacketCount: 1,
			ByteCount:   uint64(packetInfo.Length),
		}
	}
}

// Snapshot atomically swaps the active flows map in each shard with a new empty map
// and returns the old maps for processing.
func (ka *KeyedAggregator) Snapshot() []*Shard {
	oldShards := make([]*Shard, ka.shardCount)

	for i := 0; i < int(ka.shardCount); i++ {
		shard := ka.shards[i]
		shard.Mu.Lock()

		oldFlows := shard.Flows
		shard.Flows = make(map[string]*Flow) // Start using the new map

		shard.Mu.Unlock()

		// The old data is now available without a lock for processing
		oldShards[i] = &Shard{
			Flows: oldFlows,
		}
	}
	return oldShards
}