package a800mon

import (
	"context"
	"fmt"
)

const (
	topbarTitle      = "Atari800 Monitor"
	topbarCopyright  = "(c) 2026 Marcin Nowak"
	topbarRightWidth = 43
)

type TopBar struct {
	BaseVisualComponent
	lastSnapshot string
}

func NewTopBar(window *Window) *TopBar {
	return &TopBar{BaseVisualComponent: NewBaseVisualComponent(window)}
}

func (t *TopBar) Update(_ctx context.Context) (bool, error) {
	st := State()
	snap := fmt.Sprintf("%s|%t|%d|%d|%d", st.LastRPCError, st.Crashed, st.EmuMS, st.ResetMS, st.MonitorFrameTimeMS)
	if t.lastSnapshot == snap {
		return false, nil
	}
	t.lastSnapshot = snap
	return true, nil
}

func (t *TopBar) Render(_force bool) {
	st := State()
	w := t.Window()
	w.Cursor(0, 0)
	if st.LastRPCError != "" {
		w.Print(topbarTitle+" ", ColorTopbar.Attr(), false)
		w.Print(" "+st.LastRPCError+" ", ColorError.Attr(), false)
		w.FillToEOL(' ', ColorError.Attr())
	} else {
		w.Print(topbarTitle+"     "+topbarCopyright, ColorTopbar.Attr(), false)
		w.FillToEOL(' ', ColorTopbar.Attr())
	}
	start := w.Width() - topbarRightWidth
	if start < 0 {
		start = 0
	}
	w.Cursor(start, 0)
	segments := [][2]any{
		{crashLabel(st.Crashed), crashColor(st.Crashed)},
		{" UP ", ColorText},
		{fmt.Sprintf(" %s ", formatHMS(st.EmuMS)), ColorTopbar},
		{" RS ", ColorText},
		{fmt.Sprintf(" %s ", formatHMS(st.ResetMS)), ColorTopbar},
		{fmt.Sprintf(" %3d ms ", st.MonitorFrameTimeMS), ColorText},
	}
	for _, segment := range segments {
		text := segment[0].(string)
		color := segment[1].(Color)
		w.Print(text, color.Attr(), false)
	}
}

func crashLabel(crashed bool) string {
	if crashed {
		return " CRASH "
	}
	return "       "
}

func crashColor(crashed bool) Color {
	if crashed {
		return ColorError
	}
	return ColorTopbar
}
