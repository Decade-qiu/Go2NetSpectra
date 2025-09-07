package flowaggregator

import (
	"Go2NetSpectra/internal/core/model"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"sync"
)

const defaultShardCount = 256

// Shard is a part of a sharded map, containing its own map and a mutex.
type Shard struct {
	flows map[string]*model.Flow
	mu    sync.RWMutex
}

// KeyedAggregator performs aggregation for a specific set of key fields using a sharded map
// for improved concurrency.
type KeyedAggregator struct {
	Name        string
	KeyFields   []string
	shards      []*Shard
	shardCount  uint32
}

// NewKeyedAggregator creates a new sharded aggregator.
func NewKeyedAggregator(name string, keyFields []string) *KeyedAggregator {
	agg := &KeyedAggregator{
		Name:        name,
		KeyFields:   keyFields,
		shards:      make([]*Shard, defaultShardCount),
		shardCount:  defaultShardCount,
	}
	for i := 0; i < int(defaultShardCount); i++ {
		agg.shards[i] = &Shard{
			flows: make(map[string]*model.Flow),
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
	// This function remains the same
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
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if flow, ok := shard.flows[key]; ok {
		// Flow exists, update it
		flow.EndTime = packetInfo.Timestamp
		flow.PacketCount++
		flow.ByteCount += uint64(packetInfo.Length)
	} else {
		// Flow does not exist, create a new one
		shard.flows[key] = &model.Flow{
			Key:         key,
			StartTime:   packetInfo.Timestamp,
			EndTime:     packetInfo.Timestamp,
			PacketCount: 1,
			ByteCount:   uint64(packetInfo.Length),
		}
	}
}

// GetFlowCount returns the total number of active flows in the aggregator.
// Note: This is for testing/metrics purposes.
func (ka *KeyedAggregator) GetFlowCount() int {
	count := 0
	for i := 0; i < int(ka.shardCount); i++ {
		shard := ka.shards[i]
		shard.mu.RLock()
		count += len(shard.flows)
		shard.mu.RUnlock()
	}
	return count
}

// GetFlow returns a copy of a flow for a given key.
// Note: This is for testing/metrics purposes.
func (ka *KeyedAggregator) GetFlow(key string) (*model.Flow, bool) {
    shard := ka.getShard(key)
    shard.mu.RLock()
    defer shard.mu.RUnlock()
    if flow, ok := shard.flows[key]; ok {
        // Return a copy to avoid race conditions
        flowCopy := *flow
        return &flowCopy, true
    }
    return nil, false
}
