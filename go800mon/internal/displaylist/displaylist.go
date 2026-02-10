package displaylist

import "fmt"

const (
	DMACTLAddr   uint16 = 0x022F
	DMACTLHWAddr uint16 = 0xD400
	DLPTRSAddr   uint16 = 0x0230
)

type Entry struct {
	Addr    uint16
	Command byte
	Arg     uint16
}

func (e Entry) IsDLI() bool {
	return e.Command&0x80 != 0
}

func (e Entry) Mode() byte {
	return e.Command & 0x0F
}

func (e Entry) CommandName() string {
	mode := e.Mode()
	if mode == 0 {
		return "BLANK"
	}
	if mode == 1 {
		if e.Command&0x40 != 0 {
			return "JVB"
		}
		return "JMP"
	}
	return fmt.Sprintf("MODE %d", mode)
}

func (e Entry) Description() string {
	mode := e.Mode()
	prefix := ""
	if e.IsDLI() {
		prefix = "DLI "
	}
	if mode == 0 {
		count := int((e.Command>>4)&0x07) + 1
		return fmt.Sprintf("%s%d %s", prefix, count, e.CommandName())
	}
	if mode == 1 {
		return fmt.Sprintf("%s%s %04X", prefix, e.CommandName(), e.Arg)
	}
	parts := make([]string, 0, 4)
	if e.Command&0x40 != 0 {
		parts = append(parts, fmt.Sprintf("LMS %04X", e.Arg))
	}
	if e.Command&0x20 != 0 {
		parts = append(parts, "VSCROL")
	}
	if e.Command&0x10 != 0 {
		parts = append(parts, "HSCROL")
	}
	parts = append(parts, e.CommandName())
	if len(prefix) == 0 {
		return join(parts)
	}
	return prefix + join(parts)
}

type DisplayList struct {
	StartAddr uint16
	Entries   []Entry
}

func Decode(startAddr uint16, data []byte) DisplayList {
	entries := make([]Entry, 0, len(data)/2)
	pc := 0
	for pc < len(data) {
		addr := uint16((int(startAddr) + pc) & 0xFFFF)
		ir := data[pc]
		pc++
		cmd := ir & 0x0F
		arg := uint16(0)
		if cmd == 1 {
			if pc+1 >= len(data) {
				break
			}
			arg = uint16(data[pc]) | (uint16(data[pc+1]) << 8)
			pc += 2
		} else if cmd != 0 && (ir&0x40) != 0 {
			if pc+1 >= len(data) {
				break
			}
			arg = uint16(data[pc]) | (uint16(data[pc+1]) << 8)
			pc += 2
		}
		entries = append(entries, Entry{Addr: addr, Command: ir, Arg: arg})
		if cmd == 1 && (ir&0x40) != 0 {
			break
		}
	}
	return DisplayList{StartAddr: startAddr, Entries: entries}
}

type Compacted struct {
	Count int
	Entry Entry
}

func (d DisplayList) Compacted() []Compacted {
	if len(d.Entries) == 0 {
		return nil
	}
	out := make([]Compacted, 0, len(d.Entries))
	run := d.Entries[0]
	count := 1
	for i := 1; i < len(d.Entries); i++ {
		e := d.Entries[i]
		if e.Command == run.Command && e.Arg == run.Arg {
			count++
			continue
		}
		out = append(out, Compacted{Count: count, Entry: run})
		run = e
		count = 1
	}
	out = append(out, Compacted{Count: count, Entry: run})
	return out
}

type Segment struct {
	Start int
	End   int
	Mode  byte
}

func (d DisplayList) ScreenSegments(dmactl byte) []Segment {
	rows := NewMemoryMapper(d, dmactl, 4096).RowRangesWithModes()
	if len(rows) == 0 {
		return nil
	}
	segs := make([]Segment, 0, len(rows)*2)
	for _, r := range rows {
		if r.Addr == nil || r.Length == 0 {
			continue
		}
		start := int(*r.Addr)
		end := start + r.Length
		if end <= 0x10000 {
			segs = append(segs, Segment{Start: start, End: end, Mode: r.Mode})
			continue
		}
		segs = append(segs, Segment{Start: start, End: 0x10000, Mode: r.Mode})
		segs = append(segs, Segment{Start: 0, End: end & 0xFFFF, Mode: r.Mode})
	}
	if len(segs) == 0 {
		return nil
	}
	merged := make([]Segment, 0, len(segs))
	cur := segs[0]
	for i := 1; i < len(segs); i++ {
		s := segs[i]
		if s.Mode == cur.Mode && cur.Start <= s.Start && s.Start <= cur.End {
			if s.End > cur.End {
				cur.End = s.End
			}
			continue
		}
		merged = append(merged, cur)
		cur = s
	}
	merged = append(merged, cur)
	return merged
}

type RowRange struct {
	Addr   *uint16
	Length int
	Mode   byte
}

type FetchRange struct {
	Start int
	End   int
}

type RowSlice struct {
	Addr   uint16
	Length int
}

type MemoryMapper struct {
	dlist   DisplayList
	dmactl  byte
	maxRead int
}

func NewMemoryMapper(dlist DisplayList, dmactl byte, maxRead int) MemoryMapper {
	return MemoryMapper{dlist: dlist, dmactl: dmactl, maxRead: maxRead}
}

func (m MemoryMapper) widthBytes() int {
	switch m.dmactl & 0x03 {
	case 0:
		return 40
	case 1:
		return 32
	case 2:
		return 40
	case 3:
		return 48
	default:
		return 0
	}
}

func (m MemoryMapper) hscrolWidthBytes(width int) int {
	if width <= 32 {
		return 40
	}
	if width <= 40 {
		return 48
	}
	return 48
}

func (m MemoryMapper) bytesPerLine(mode byte, width int) int {
	if mode == 0 || mode == 1 {
		return 0
	}
	switch mode {
	case 2, 3, 4, 5, 0xD, 0xE, 0xF:
		return width
	case 6, 7, 0xA, 0xB, 0xC:
		return width / 2
	case 8, 9:
		return width / 4
	}
	return width
}

func (m MemoryMapper) BytesPerLine(mode byte) int {
	return m.bytesPerLine(mode, m.widthBytes())
}

func (m MemoryMapper) RowRanges() []RowRange {
	width := m.widthBytes()
	var addr *uint16
	rows := make([]RowRange, 0, len(m.dlist.Entries))
	for _, e := range m.dlist.Entries {
		ir := e.Command
		mode := ir & 0x0F
		if mode == 0 {
			count := int((ir>>4)&0x07) + 1
			for i := 0; i < count; i++ {
				rows = append(rows, RowRange{Addr: nil, Length: 0, Mode: mode})
			}
			continue
		}
		if mode == 1 {
			if ir&0x40 != 0 {
				break
			}
			continue
		}
		if ir&0x40 != 0 {
			v := e.Arg
			addr = &v
		}
		if addr == nil {
			continue
		}
		lineWidth := width
		if ir&0x10 != 0 {
			lineWidth = m.hscrolWidthBytes(width)
		}
		n := m.bytesPerLine(mode, lineWidth)
		v := *addr
		rows = append(rows, RowRange{Addr: &v, Length: n, Mode: mode})
		next := uint16((int(v) + n) & 0xFFFF)
		addr = &next
	}
	return rows
}

func (m MemoryMapper) RowRangesWithModes() []RowRange {
	rows := m.RowRanges()
	out := rows[:0]
	for _, r := range rows {
		if r.Mode == 0 {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (m MemoryMapper) Plan() ([]FetchRange, []RowSlice) {
	rows := m.RowRanges()
	type segment struct{ s, e int }
	segments := make([]segment, 0, len(rows)*2)
	for _, r := range rows {
		if r.Addr == nil || r.Length == 0 {
			continue
		}
		start := int(*r.Addr)
		end := start + r.Length
		if end <= 0x10000 {
			segments = append(segments, segment{s: start, e: end})
		} else {
			segments = append(segments, segment{s: start, e: 0x10000})
			segments = append(segments, segment{s: 0, e: end & 0xFFFF})
		}
	}
	rowSlices := make([]RowSlice, 0, len(rows))
	for _, r := range rows {
		if r.Addr == nil || r.Length == 0 {
			continue
		}
		rowSlices = append(rowSlices, RowSlice{Addr: *r.Addr, Length: r.Length})
	}
	if len(segments) == 0 {
		return nil, rowSlices
	}

	for i := 1; i < len(segments); i++ {
		j := i
		for j > 0 && segments[j-1].s > segments[j].s {
			segments[j-1], segments[j] = segments[j], segments[j-1]
			j--
		}
	}
	merged := make([]segment, 0, len(segments))
	cur := segments[0]
	for i := 1; i < len(segments); i++ {
		s := segments[i]
		if s.s <= cur.e {
			if s.e > cur.e {
				cur.e = s.e
			}
			continue
		}
		merged = append(merged, cur)
		cur = s
	}
	merged = append(merged, cur)

	fetch := make([]FetchRange, 0, len(merged))
	for _, seg := range merged {
		start := seg.s
		for start < seg.e {
			end := seg.e
			if m.maxRead > 0 && start+m.maxRead < end {
				end = start + m.maxRead
			}
			fetch = append(fetch, FetchRange{Start: start, End: end})
			start = end
		}
	}
	return fetch, rowSlices
}

func join(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += " " + parts[i]
	}
	return out
}
