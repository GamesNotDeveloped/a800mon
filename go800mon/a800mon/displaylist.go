package a800mon

import (
	"context"
	"fmt"

	dl "go800mon/internal/displaylist"
)

const (
	DMACTLAddr   = dl.DMACTLAddr
	DMACTLHWAddr = dl.DMACTLHWAddr
	DLPTRSAddr   = dl.DLPTRSAddr
)

func DecodeDisplayList(startAddr uint16, data []byte) dl.DisplayList {
	return dl.Decode(startAddr, data)
}

type DisplayListViewer struct {
	BaseVisualComponent
	rpc          *RpcClient
	lastSnapshot string
}

func NewDisplayListViewer(rpc *RpcClient, window *Window) *DisplayListViewer {
	return &DisplayListViewer{BaseVisualComponent: NewBaseVisualComponent(window), rpc: rpc}
}

func (v *DisplayListViewer) Update(ctx context.Context) (bool, error) {
	if State().InputFocus {
		return false, nil
	}
	startAddr, err := v.rpc.ReadVector(ctx, DLPTRSAddr)
	if err != nil {
		return false, nil
	}
	dump, err := v.rpc.ReadDisplayList(ctx)
	if err != nil {
		return false, nil
	}
	dmactl, err := v.rpc.ReadByte(ctx, DMACTLAddr)
	if err != nil {
		return false, nil
	}
	if (dmactl & 0x03) == 0 {
		if hw, err := v.rpc.ReadByte(ctx, DMACTLHWAddr); err == nil {
			dmactl = hw
		}
	}
	dlist := DecodeDisplayList(startAddr, dump)
	store.setDList(dlist, dmactl)
	st := State()
	if st.DisplayListInspect {
		segs := dlist.ScreenSegments(dmactl)
		if len(segs) == 0 {
			store.setDListSelectedRegion(nil)
		} else if st.DListSelectedRegion == nil {
			idx := 0
			store.setDListSelectedRegion(&idx)
		} else if *st.DListSelectedRegion >= len(segs) {
			idx := len(segs) - 1
			store.setDListSelectedRegion(&idx)
		}
	}
	snapshot := fmt.Sprintf("%04X|%02X|%d", startAddr, dmactl, len(dlist.Entries))
	if len(dlist.Entries) > 0 {
		first := dlist.Entries[0]
		last := dlist.Entries[len(dlist.Entries)-1]
		snapshot += fmt.Sprintf("|%02X-%04X|%02X-%04X", first.Command, first.Arg, last.Command, last.Arg)
	}
	if v.lastSnapshot == snapshot {
		return false, nil
	}
	v.lastSnapshot = snapshot
	return true, nil
}

func (v *DisplayListViewer) Render(_force bool) {
	st := State()
	w := v.Window()
	if st.DisplayListInspect {
		segs := st.DList.ScreenSegments(st.DMACTL)
		if len(segs) == 0 {
			w.ClearToBottom()
			return
		}
		w.Cursor(0, 0)
		for idx, seg := range segs {
			length := seg.End - seg.Start
			last := (seg.End - 1) & 0xFFFF
			attr := 0
			selected := st.DListSelectedRegion != nil && idx == *st.DListSelectedRegion
			if selected {
				attr = AttrReverse()
			}
			w.PrintLine(fmt.Sprintf("%04X-%04X len=%04X antic=%d", seg.Start&0xFFFF, last, length, seg.Mode), attr, false)
			if selected {
				w.ClearToEOL(true)
			} else {
				w.ClearToEOL(false)
			}
		}
		w.ClearToBottom()
		return
	}
	w.Cursor(0, 0)
	for _, c := range st.DList.Compacted() {
		addr := fmt.Sprintf("%04X:", c.Entry.Addr)
		desc := " " + c.Entry.Description()
		if c.Count > 1 {
			desc = fmt.Sprintf(" %dx %s", c.Count, c.Entry.Description())
		}
		w.Print(addr, ColorAddress.Attr(), false)
		w.Print(desc, ColorText.Attr(), false)
		w.ClearToEOL(false)
		w.Newline()
	}
	w.ClearToBottom()
}
