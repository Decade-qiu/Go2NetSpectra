package sketch

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/impl/sketch/statistic"
	"Go2NetSpectra/internal/factory"
	"Go2NetSpectra/internal/model"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- Factory Registration ---

func init() {
	factory.RegisterAggregator("sketch", func(cfg *config.Config) (*factory.TaskGroup, error) {
		sketchCfg := cfg.Aggregator.Sketch

		// Create all enabled writers for this aggregator group
		writers := make([]model.Writer, 0, len(sketchCfg.Writers))
		for _, writerDef := range sketchCfg.Writers {
			if !writerDef.Enabled {
				continue
			}

			interval, err := time.ParseDuration(writerDef.SnapshotInterval)
			if err != nil {
				log.Printf("Warning: invalid snapshot_interval for writer type '%s': %v, skipping.", writerDef.Type, err)
				continue
			}

			var writer model.Writer
			switch writerDef.Type {
			case "text":
				writer = NewTextWriter(writerDef.Text.RootPath, interval)
				log.Printf("Text writer created at %s", writerDef.Text.RootPath)
			case "clickhouse":
				writer, err = NewClickHouseWriter(writerDef.ClickHouse, interval)
				if err != nil {
					log.Printf("Warning: failed to create writer type '%s': %v, skipping.", writerDef.Type, err)
					continue
				} else {
					log.Printf("ClickHouse writer created for database %s at %s:%d", writerDef.ClickHouse.Database, writerDef.ClickHouse.Host, writerDef.ClickHouse.Port)
				}
			default:
				log.Printf("Warning: unknown writer type '%s' in sketch aggregator config, skipping.", writerDef.Type)
				continue
			}
			writers = append(writers, writer)
		}

		// Create all tasks for this aggregator group
		tasks := make([]model.Task, len(sketchCfg.Tasks))
		for i, taskCfg := range sketchCfg.Tasks {
			tasks[i] = New(taskCfg)
		}

		return &factory.TaskGroup{Tasks: tasks, Writers: writers}, nil
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

// New creates a new Sketch task based on the provided configuration.
func New(cfg config.SketchTaskDef) model.Task {
	flowSize := uint32(0)
	for _, f := range cfg.FlowFields {
		flowSize += fieldByteSize(f)
	}
	elemSize := uint32(0)
	for _, f := range cfg.ElementFields {
		elemSize += fieldByteSize(f)
	}

	var sketchImpl statistic.Sketch
	switch cfg.SktType {
	case 0: // CountMin
		log.Printf("Creating CountMin Sketch '%s' for:\n\tflow fields %v (bytes %d)\n\telement fields %v (bytes %d) with width %d, depth %d, size_thereshold %d, count_thereshold %d\n",
			cfg.Name, cfg.FlowFields, flowSize, cfg.ElementFields, elemSize, cfg.Width, cfg.Depth, cfg.SizeThereshold, cfg.CountThereshold)
		sketchImpl = statistic.NewCountMin(cfg.Width, cfg.Depth, cfg.SizeThereshold, cfg.CountThereshold, flowSize)
	case 1: // SuperSpread
		log.Printf("Creating SuperSpread Sketch '%s' for:\n\tflow fields %v (bytes %d)\n\telement fields %v (bytes %d) with width %d, depth %d, threshold %d, m %d, size %d, base %.2f, b %.2f\n",
			cfg.Name, cfg.FlowFields, flowSize, cfg.ElementFields, elemSize, cfg.Width, cfg.Depth, cfg.CountThereshold, cfg.M, cfg.Size, cfg.Base, cfg.B)
		sketchImpl = statistic.NewSuperSpread(cfg.Width, cfg.Depth, cfg.CountThereshold, cfg.M, cfg.Size, cfg.Base, cfg.B, flowSize)
	default:
		log.Fatalf("Unknown sketch type: %d for task %s", cfg.SktType, cfg.Name)
	}

	return &Task{
		name:          cfg.Name,
		flowFields:    cfg.FlowFields,
		elementFields: cfg.ElementFields,
		flowSize:      flowSize,
		elemSize:      elemSize,
		sketch:        sketchImpl,
	}
}

// Name returns the name of the task.
func (t *Task) Name() string {
	return t.name
}

// Fields 
func (t *Task) Fields() []string {
	return t.flowFields
}

// Func 
func (t *Task) DecodeFlowFunc() func(flow []byte, fields []string) string {
	return t.DecodeFlow
}

// ProcessPacket processes a single packet, creating or updating a flow in the correct shard.
func (t *Task) ProcessPacket(packetInfo *model.PacketInfo) {
	flow := flowPool.Get().([]byte)[:t.flowSize]
    elem := elemPool.Get().([]byte)[:t.elemSize]
	defer flowPool.Put(flow)
 	defer elemPool.Put(elem)

	err := t.generateFlowAndElem(flow, elem, &packetInfo.FiveTuple)
	if err != nil {
		log.Printf("Error generating key for task '%s': %v\n", t.name, err)
		return
	}

	t.sketch.Insert(flow, elem, uint32(packetInfo.Length))
}

func (t *Task) Query(flow []byte) uint64 {
	return t.sketch.Query(flow)
}

func (t *Task) Snapshot() any {
	return t.sketch.HeavyHitters()
}

// Reset clears the internal state of the task, preparing for a new measurement period.
func (t *Task) Reset() {
	t.sketch.Reset()
}

func (t *Task) AlerterMsg(rules []config.AlerterRule) string {
	// Perform a snapshot to get the latest data for evaluation.
	snapshotData, ok := t.Snapshot().(statistic.HeavyRecord)
	if !ok {
		log.Printf("ERROR: AlerterMsg in sketch task received unexpected snapshot type: %T", t.sketch.HeavyHitters())
		return ""
	}

	var triggeredMessages []string

	for _, rule := range rules {
		if rule.TaskName != t.name {
			continue
		}

		var hitters []string
		switch rule.Metric {
		case "heavy_hitter_count":
			for _, hitter := range snapshotData.Count {
				if check(float64(hitter.Count), rule.Threshold, rule.Operator) {
					hitters = append(hitters, fmt.Sprintf("<tr><td><code>%s</code></td><td>%d</td></tr>", t.DecodeFlow(hitter.Flow, t.flowFields), hitter.Count))
				}
			}
		case "heavy_hitter_size":
			for _, hitter := range snapshotData.Size {
				if check(float64(hitter.Size), rule.Threshold, rule.Operator) {
					hitters = append(hitters, fmt.Sprintf("<tr><td><code>%s</code></td><td>%d bytes</td></tr>", t.DecodeFlow(hitter.Flow, t.flowFields), hitter.Size))
				}
			}
		case "super_spreader_spread":
			if snapshotData.Size == nil {
				for _, spreader := range snapshotData.Count {
					if check(float64(spreader.Count), rule.Threshold, rule.Operator) {
						hitters = append(hitters, fmt.Sprintf("<tr><td><code>%s</code></td><td>%d</td></tr>", t.DecodeFlow(spreader.Flow, t.flowFields), spreader.Count))
					}
				}
			}
		}

		if len(hitters) > 0 {
			itemsTable := fmt.Sprintf("<table border=\"1\" cellpadding=\"5\" cellspacing=\"0\">"+
				"<tr><th>Flow/Source</th><th>Value</th></tr>%s</table>", strings.Join(hitters, ""))

			msg := fmt.Sprintf("<h3>Alert: %s</h3>"+
				"<ul>"+
				"<li><b>Task:</b> <code>%s</code></li>"+
				"<li><b>Metric:</b> <code>%s</code></li>"+
				"<li><b>Condition:</b> <code>%s %.2f</code></li>"+
				"</ul>"+
				"<p><b>Triggering Items:</b></p>%s",
				rule.Name, rule.TaskName, rule.Metric, rule.Operator, rule.Threshold, itemsTable)
			triggeredMessages = append(triggeredMessages, msg)
		}
	}

	return strings.Join(triggeredMessages, "<br><hr><br>")
}

// check compares a value against a threshold based on an operator.
func check(value, threshold float64, operator string) bool {
	switch operator {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case "=":
		return value == threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	default:
		log.Printf("Warning: unknown operator '%s' in alerter rule", operator)
		return false
	}
}

// generateFlowAndElem creates Flow and Element keys based on the configured fields.
func (t *Task) generateFlowAndElem(flow, elem []byte, ft *model.FiveTuple) (error) {
	offset := 0
	for _, f := range t.flowFields {
		offset = t.EncodeFlow(flow, offset, f, ft)
	}
	offset = 0
	for _, f := range t.elementFields {
		offset = t.EncodeFlow(elem, offset, f, ft)
	}

	return nil
}

func (t *Task) EncodeFlow(buf []byte, offset int, field string, ft *model.FiveTuple) int {
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

func (t *Task) DecodeFlow(flow []byte, fields []string) string {
    var parts []string
    offset := 0

    for _, f := range fields {
        switch f {
        case "SrcIP", "DstIP":
            ip := net.IP(flow[offset : offset+IPv6ByteSize])
            parts = append(parts, ip.String())
            offset += IPv6ByteSize
        case "SrcPort", "DstPort":
            port := binary.BigEndian.Uint16(flow[offset : offset+PortByteSize])
            parts = append(parts, strconv.Itoa(int(port)))
            offset += PortByteSize
        case "Protocol":
            proto := uint8(flow[offset])
            parts = append(parts, strconv.Itoa(int(proto)))
            offset += ProtoByteSize
        }
    }

    return strings.Join(parts, " ")
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
