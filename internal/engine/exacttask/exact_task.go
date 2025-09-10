package exacttask

import (
	"Go2NetSpectra/internal/engine/exacttask/statistic"
	"Go2NetSpectra/internal/model"
	"fmt"
	"hash/fnv"
	"log"
	"strconv"
	"strings"
)

// --- Data Structures ---

// --- Task Implementation ---

const defaultShardCount = 256

// Task performs exact aggregation for a specific set of key fields using a sharded map.
// It implements the model.Task interface.
type Task struct {
	name        string
	keyFields   []string
	shards      []*statistic.Shard
	shardCount  uint32
}

// New creates a new exact aggregation task.
func New(name string, keyFields []string, numShards uint32) model.Task {
	if numShards <= 0 || numShards >= 32768 {
		numShards = defaultShardCount
	}
	log.Printf("Creating ExactTask '%s' with %d shards for keys: %v", name, numShards, keyFields)
	task := &Task{
		name:        name,
		keyFields:   keyFields,
		shards:      make([]*statistic.Shard, numShards),
		shardCount:  numShards,
	}
	for i := 0; i < int(numShards); i++ {
		task.shards[i] = &statistic.Shard{
			Flows: make(map[string]*statistic.Flow),
		}
	}
	return task
}

// Name returns the name of the task.
func (t *Task) Name() string {
	return t.name
}

// ProcessPacket processes a single packet, creating or updating a flow in the correct shard.
func (t *Task) ProcessPacket(packetInfo *model.PacketInfo) {
	key, err := t.generateKey(packetInfo.FiveTuple)
	if err != nil {
		log.Printf("Error generating key for task '%s': %v\n", t.name, err)
		return
	}

	shard := t.getShard(key)
	shard.Mu.Lock()
	defer shard.Mu.Unlock()

	if flow, ok := shard.Flows[key]; ok {
		flow.EndTime = packetInfo.Timestamp
		flow.PacketCount++
		flow.ByteCount += uint64(packetInfo.Length)
	} else {
		shard.Flows[key] = &statistic.Flow{
			Key:         key,
			StartTime:   packetInfo.Timestamp,
			EndTime:     packetInfo.Timestamp,
			PacketCount: 1,
			ByteCount:   uint64(packetInfo.Length),
		}
	}
}

// Snapshot returns the current data and resets the internal state.
func (t *Task) Snapshot() interface{} {
	oldShards := make([]*statistic.Shard, t.shardCount)
	for i := 0; i < int(t.shardCount); i++ {
		shard := t.shards[i]
		shard.Mu.Lock()

		oldFlows := shard.Flows
		shard.Flows = make(map[string]*statistic.Flow) // Reset with a new map

		shard.Mu.Unlock()

		oldShards[i] = &statistic.Shard{
			Flows: oldFlows,
		}
	}
	return statistic.SnapshotData{
		TaskName: t.name,
		Shards:   oldShards,
	}
}

// getShard returns the appropriate shard for a given key.
func (t *Task) getShard(key string) *statistic.Shard {
	hasher := fnv.New32a()
	hasher.Write([]byte(key))
	return t.shards[hasher.Sum32()%t.shardCount]
}

// generateKey creates a unique string key for a packet based on the task's KeyFields.
func (t *Task) generateKey(ft model.FiveTuple) (string, error) {
	var parts []string
	for _, field := range t.keyFields {
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
