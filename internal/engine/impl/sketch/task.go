package sketch

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/impl/sketch/statistic"
	"Go2NetSpectra/internal/factory"
	"Go2NetSpectra/internal/model"
	"log"
	"sync"
)

// --- Factory Registration ---

func init() {
	factory.RegisterAggregator("sketch", func(cfg *config.Config) ([]model.Task, []model.Writer, error) {
		exactCfg := cfg.Aggregator.Sketch

		// Create all enabled writers for this aggregator group
		// No writers implemented yet
		writers := make([]model.Writer, 0)

		// Create all tasks for this aggregator group
		tasks := make([]model.Task, len(exactCfg.Tasks))
		for i, taskCfg := range exactCfg.Tasks {
			tasks[i] = New(taskCfg.Name, taskCfg.FlowFields, taskCfg.ElementFields, taskCfg.Width, taskCfg.Depth, taskCfg.SizeThereshold, taskCfg.CountThereshold)
		}

		return tasks, writers, nil
	})
}

// --- Task Implementation ---

const (
	IPv4ByteSize = 4
	IPv6ByteSize = 16
	PortByteSize = 2
	ProtoByteSize = 1

	MaxFieldSize = 37 // IPv6(16) + IPv6(16) + Port(2) + Port(2) + Proto(1) = 37
)

var (
    flowPool = sync.Pool{
        New: func() any {
            return make([]byte, MaxFieldSize)
        },
    }
    elemPool = sync.Pool{
        New: func() any {
            return make([]byte, MaxFieldSize)
        },
    }
)

type Task struct {
	name string
	// flow key fields
	flowFields []string
	// the byte size of flow key
	flowSize uint32
	// element key fields
	elementFields []string
	// the byte size of element key
	elemSize uint32
	// data
	sketch   statistic.Sketch
}

// New creates a new Sketch task with the given name, flow fields, and element fields.
func New(name string, flowFields, elementFields []string, w, d, st, ct uint32) model.Task {
	flowSize := uint32(0)
	for _, f := range flowFields {
		flowSize += fieldByteSize(f)
	}
	elemSize := uint32(0)
	for _, f := range elementFields {
		elemSize += fieldByteSize(f)
	}

	log.Printf("Creating Sketch '%s' for:\n\tflow fields %v (bytes %d)\n\telement fields %v (bytes %d) with width %d, depth %d, size_thereshold %d, count_thereshold %d\n",
		name, flowFields, flowSize, elementFields, elemSize, w, d, st, ct)

	return &Task{
		name:          name,
		flowFields:    flowFields,
		elementFields: elementFields,
		flowSize:      flowSize,
		elemSize:      elemSize,
		sketch:        statistic.NewCountMin(w, d, st, ct, flowSize),
	}
}

// Name returns the name of the task.
func (t *Task) Name() string {
	return t.name
}

// ProcessPacket processes a single packet, creating or updating a flow in the correct shard.
func (t *Task) ProcessPacket(packetInfo *model.PacketInfo) {
	flow := flowPool.Get().([]byte)[:t.flowSize]
    elem := elemPool.Get().([]byte)[:t.elemSize]
	defer flowPool.Put(flow)
 	defer elemPool.Put(elem)

	err := t.generateFlowAndElem(flow, elem, packetInfo.FiveTuple)
	if err != nil {
		log.Printf("Error generating key for task '%s': %v\n", t.name, err)
		return
	}

	t.sketch.Insert(flow, elem, uint32(packetInfo.Length))
}

func (t *Task) Query(flow []byte) uint64 {
	return t.sketch.Query(flow)
}

func (t *Task) Snapshot() interface{} {
	return t.sketch.HeavyHitters()
}

// Reset clears the internal state of the task, preparing for a new measurement period.
func (t *Task) Reset() {
}

// generateFlowAndElem creates Flow and Element keys based on the configured fields.
func (t *Task) generateFlowAndElem(flow, elem []byte, ft model.FiveTuple) (error) {
	writeBytes := func(buf []byte, offset int, field string) int {
		switch field {
		case "SrcIP":
			copy(buf[offset:], ft.SrcIP)
			offset += IPv6ByteSize
		case "DstIP":
			copy(buf[offset:], ft.DstIP)
			offset += IPv6ByteSize
		case "SrcPort":
			buf[offset] = byte(ft.SrcPort >> 8)
			buf[offset+1] = byte(ft.SrcPort & 0xFF)
			offset += PortByteSize
		case "DstPort":
			buf[offset] = byte(ft.DstPort >> 8)
			buf[offset+1] = byte(ft.DstPort & 0xFF)
			offset += PortByteSize
		case "Protocol":
			buf[offset] = byte(ft.Protocol)
			offset += ProtoByteSize
		}
		return offset
	}

	offset := 0
	for _, f := range t.flowFields {
		offset = writeBytes(flow, offset, f)
	}
	offset = 0
	for _, f := range t.elementFields {
		offset = writeBytes(elem, offset, f)
	}

	return nil
}

func fieldByteSize(field string) uint32 {
	switch field {
	case "SrcIP", "DstIP":
		return IPv6ByteSize
	case "SrcPort", "DstPort":
		return PortByteSize
	case "Protocol":
		return ProtoByteSize
	default:
		return 0
	}
}
