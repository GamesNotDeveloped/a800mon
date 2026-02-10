package a800mon

import (
	"context"
	"fmt"
	"strings"
)

type BreakpointsViewer struct {
	BaseVisualComponent
	rpc          *RpcClient
	screen       *Screen
	dispatcher   *ActionDispatcher
	lastSnapshot string
	lastStateSeq uint64
	hasSnapshot  bool
}

func NewBreakpointsViewer(rpc *RpcClient, window *Window) *BreakpointsViewer {
	return &BreakpointsViewer{
		BaseVisualComponent: NewBaseVisualComponent(window),
		rpc:                 rpc,
	}
}

func (v *BreakpointsViewer) BindInput(screen *Screen, dispatcher *ActionDispatcher) {
	v.screen = screen
	v.dispatcher = dispatcher
}

func (v *BreakpointsViewer) Update(ctx context.Context) (bool, error) {
	st := State()
	if !st.BreakpointsSupported {
		return false, nil
	}
	if v.hasSnapshot && v.lastStateSeq == st.StateSeq {
		return false, nil
	}
	v.lastStateSeq = st.StateSeq

	list, err := v.rpc.BPList(ctx)
	if err != nil {
		return false, nil
	}
	clauses := make([]BreakpointClauseRow, 0, len(list.Clauses))
	for _, clause := range list.Clauses {
		conds := make([]BreakpointConditionRow, 0, len(clause))
		for _, cond := range clause {
			conds = append(conds, BreakpointConditionRow{
				CondType: cond.Type,
				Op:       cond.Op,
				Addr:     cond.Addr,
				Value:    cond.Value,
			})
		}
		clauses = append(clauses, BreakpointClauseRow{Conditions: conds})
	}
	snapshot := buildBreakpointsSnapshot(list.Enabled, clauses)
	if v.hasSnapshot && snapshot == v.lastSnapshot {
		return false, nil
	}
	v.hasSnapshot = true
	v.lastSnapshot = snapshot
	if v.dispatcher != nil {
		v.dispatcher.updateBreakpoints(list.Enabled, clauses)
	}
	return true, nil
}

func (v *BreakpointsViewer) Render(_force bool) {
	st := State()
	w := v.Window()
	ih := w.Height()
	if ih <= 0 {
		return
	}
	w.Cursor(0, 0)
	w.SetTagActive("bp_enabled", st.BreakpointsEnabled)

	maxRows := ih
	if len(st.Breakpoints) == 0 {
		if maxRows > 0 {
			w.Print("No breakpoint clauses.", ColorComment.Attr(), false)
			w.ClearToEOL(false)
			w.Newline()
			w.ClearToBottom()
		}
		return
	}
	drawn := 0
	for i := 0; i < len(st.Breakpoints) && i < maxRows; i++ {
		w.Print(fmt.Sprintf("#%02d ", i+1), ColorAddress.Attr(), false)
		v.printClause(st.Breakpoints[i])
		w.ClearToEOL(false)
		w.Newline()
		drawn++
	}
	if drawn < ih {
		w.ClearToBottom()
	}
}

func (v *BreakpointsViewer) HandleInput(ch int) bool {
	if v.screen == nil {
		return false
	}
	if v.screen.Focused() != v.Window() {
		return false
	}
	return false
}

func (v *BreakpointsViewer) printClause(clause BreakpointClauseRow) {
	for i, cond := range clause.Conditions {
		if i > 0 {
			v.Window().Print(" && ", ColorText.Attr(), false)
		}
		v.printCondition(cond)
	}
}

func (v *BreakpointsViewer) printCondition(cond BreakpointConditionRow) {
	op := bpOpSymbol(cond.Op)
	if cond.CondType == 9 {
		v.Window().Print("mem[", ColorText.Attr(), false)
		v.Window().Print(formatHex16(cond.Addr), ColorAddress.Attr(), false)
		v.Window().Print("]", ColorText.Attr(), false)
	} else {
		v.Window().Print(bpTypeName(cond.CondType), ColorText.Attr(), false)
	}
	v.Window().Print(" "+op+" ", ColorText.Attr(), false)
	if cond.CondType == 2 || cond.CondType == 3 || cond.CondType == 4 || cond.CondType == 5 {
		v.Window().Print(fmt.Sprintf("%02X", cond.Value), ColorAddress.Attr(), false)
		return
	}
	v.Window().Print(formatHex16(cond.Value), ColorAddress.Attr(), false)
}

func bpTypeName(condType byte) string {
	switch condType {
	case 1:
		return "pc"
	case 2:
		return "a"
	case 3:
		return "x"
	case 4:
		return "y"
	case 5:
		return "s"
	case 6:
		return "read"
	case 7:
		return "write"
	case 8:
		return "access"
	default:
		return fmt.Sprintf("type%d", condType)
	}
}

func bpOpSymbol(op byte) string {
	switch op {
	case 1:
		return "<"
	case 2:
		return "<="
	case 3:
		return "=="
	case 4:
		return "!="
	case 5:
		return ">="
	case 6:
		return ">"
	default:
		return fmt.Sprintf("op%d", op)
	}
}

func buildBreakpointsSnapshot(enabled bool, clauses []BreakpointClauseRow) string {
	parts := make([]string, 0, len(clauses)+1)
	parts = append(parts, fmt.Sprintf("enabled:%t", enabled))
	for _, clause := range clauses {
		items := make([]string, 0, len(clause.Conditions))
		for _, cond := range clause.Conditions {
			items = append(items, fmt.Sprintf("%d:%d:%04X:%04X", cond.CondType, cond.Op, cond.Addr, cond.Value))
		}
		parts = append(parts, strings.Join(items, ","))
	}
	return strings.Join(parts, "|")
}
