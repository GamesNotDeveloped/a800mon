package a800mon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/alecthomas/kong"
	"go800mon/internal/disasm"
	dl "go800mon/internal/displaylist"
	"go800mon/internal/memory"
)

type cliArgs struct {
	Socket         string            `short:"s" default:"/tmp/atari.sock" help:"Path to Atari800 monitor socket."`
	Monitor        cliEmptyCmd       `cmd:"" help:"Run the curses monitor UI."`
	Run            cliRunCmd         `cmd:"" help:"Run a file via RPC."`
	Pause          cliEmptyCmd       `cmd:"" help:"Pause emulation."`
	Step           cliEmptyCmd       `cmd:"" help:"Step one instruction."`
	StepVBL        cliEmptyCmd       `cmd:"" name:"stepvbl" help:"Step one VBLANK."`
	Continue       cliEmptyCmd       `cmd:"" name:"continue" help:"Continue emulation."`
	Coldstart      cliEmptyCmd       `cmd:"" help:"Cold start emulation."`
	Warmstart      cliEmptyCmd       `cmd:"" help:"Warm start emulation."`
	Removecartrige cliEmptyCmd       `cmd:"" help:"Remove cartridge."`
	Removetape     cliEmptyCmd       `cmd:"" help:"Remove cassette."`
	Removedisks    cliEmptyCmd       `cmd:"" help:"Remove all disks."`
	Emulator       cliEmulatorCmd    `cmd:"" help:"Emulator control commands."`
	BP             cliBreakpointsCmd `cmd:"" name:"bp" help:"Manage user breakpoints."`
	DList          cliDListCmd       `cmd:"" name:"dlist" help:"Dump display list."`
	CPUState       cliEmptyCmd       `cmd:"" name:"cpustate" help:"Show CPU state."`
	History        cliHistoryCmd     `cmd:"" help:"Show CPU execution history."`
	Status         cliEmptyCmd       `cmd:"" help:"Get status."`
	Ping           cliEmptyCmd       `cmd:"" help:"Ping RPC server."`
	ReadMem        cliReadMemCmd     `cmd:"" name:"readmem" help:"Read memory."`
	WriteMem       cliWriteMemCmd    `cmd:"" name:"writemem" help:"Write memory."`
	Disasm         cliDisasmCmd      `cmd:"" help:"Disassemble 6502 memory."`
	Screen         cliScreenCmd      `cmd:"" help:"Dump screen memory segments."`
}

type cliEmptyCmd struct{}

type cliRunCmd struct {
	Path string `arg:"" help:"Path to file."`
}

type cliHistoryCmd struct {
	Count int `short:"n" name:"count" default:"-1" help:"Limit output to last N entries."`
}

type cliEmulatorCmd struct {
	Stop     cliEmptyCmd `cmd:"" help:"Stop emulator."`
	Restart  cliEmptyCmd `cmd:"" help:"Restart emulator."`
	Features cliEmptyCmd `cmd:"" help:"Show emulator capability flags."`
	Debug    cliEmptyCmd `cmd:"" help:"Switch to emulator built-in monitor."`
}

type cliBreakpointsCmd struct {
	LS    cliEmptyCmd    `cmd:"" default:"1" name:"ls" help:"List user breakpoint clauses."`
	Add   cliBPAddCmd    `cmd:"" help:"Add one breakpoint clause (AND)."`
	Del   cliBPDeleteCmd `cmd:"" name:"del" help:"Delete clause by index (1-based)."`
	Clear cliEmptyCmd    `cmd:"" help:"Clear all breakpoint clauses."`
	On    cliEmptyCmd    `cmd:"" name:"on" help:"Enable all user breakpoints."`
	Off   cliEmptyCmd    `cmd:"" name:"off" help:"Disable all user breakpoints."`
}

type cliBPAddCmd struct {
	Conditions []string `arg:"" help:"Conditions joined by AND in one clause."`
}

type cliBPDeleteCmd struct {
	Index int `arg:"" help:"Clause index (1-based)."`
}

type cliDListCmd struct {
	Address *string `arg:"" optional:"" help:"Optional display list start address (hex: 0xNNNN, $NNNN, NNNN)."`
}

type cliReadMemCmd struct {
	Addr    string `arg:"" help:"Address (hex: 0xNNNN, $NNNN, NNNN)."`
	Length  string `arg:"" help:"Length (hex: 0xNNNN, $NNNN, NNNN)."`
	Raw     bool   `name:"raw" xor:"format" help:"Output raw bytes without formatting."`
	JSON    bool   `name:"json" xor:"format" help:"Output JSON with address and buffer."`
	ATASCII bool   `short:"a" name:"atascii" help:"Render ASCII column using ATASCII mapping."`
	Columns *int   `short:"c" name:"columns" help:"Bytes per line (default: 16)."`
	NoHex   bool   `name:"nohex" help:"Hide hex column in formatted output."`
	NoASCII bool   `name:"noascii" help:"Hide ASCII column in formatted output."`
}

type cliWriteMemCmd struct {
	Addr    string   `arg:"" help:"Address (hex: 0xNNNN, $NNNN, NNNN)."`
	Bytes   []string `arg:"" optional:"" help:"Byte/word values (hex). Values > FF are written as little-endian words."`
	Hex     *string  `name:"hex" help:"Hex payload (001122...) or '-' to read from stdin."`
	Text    *string  `name:"text" help:"Text payload or '-' to read from stdin."`
	ATASCII bool     `short:"a" name:"atascii" help:"Encode --text using ATASCII bytes."`
	Screen  bool     `short:"S" name:"screen" help:"Convert payload from ATASCII to screen codes."`
}

type cliDisasmCmd struct {
	Addr   string `arg:"" help:"Address (hex: 0xNNNN, $NNNN, NNNN)."`
	Length string `arg:"" help:"Length (hex: 0xNNNN, $NNNN, NNNN)."`
}

type cliScreenCmd struct {
	Segment *int `arg:"" optional:"" help:"Segment number (1-based). When omitted, dumps all segments."`
	List    bool `short:"l" name:"list" help:"List screen segments."`
	Raw     bool `name:"raw" xor:"format" help:"Output raw bytes without formatting."`
	JSON    bool `name:"json" xor:"format" help:"Output JSON with address and buffer."`
	ATASCII bool `short:"a" name:"atascii" help:"Render ASCII column using ATASCII mapping."`
	Columns *int `short:"c" name:"columns" help:"Bytes per line (default: 16)."`
	NoHex   bool `name:"nohex" help:"Hide hex column in formatted output."`
	NoASCII bool `name:"noascii" help:"Hide ASCII column in formatted output."`
}

func Main(argv []string) int {
	if len(argv) == 0 {
		return cmdMonitor("/tmp/atari.sock")
	}
	args, parsed, err := parseCLI(argv)
	if err != nil && canFallbackToMonitor(argv) {
		fallbackArgv := append(append([]string{}, argv...), "monitor")
		args, parsed, err = parseCLI(fallbackArgv)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, formatCliError(err))
		return 2
	}
	socket := args.Socket
	selected := parsed.Selected()
	if selected == nil {
		return cmdMonitor(socket)
	}
	switch selected.Path() {
	case "monitor":
		return cmdMonitor(socket)
	case "run":
		return cmdRun(socket, args.Run.Path)
	case "pause":
		return cmdSimple(socket, CmdPause)
	case "step":
		return cmdStepLike(socket, CmdStep)
	case "stepvbl":
		return cmdStepLike(socket, CmdStepVBlank)
	case "continue":
		return cmdSimple(socket, CmdContinue)
	case "coldstart":
		return cmdSimple(socket, CmdColdstart)
	case "warmstart":
		return cmdSimple(socket, CmdWarmstart)
	case "removecartrige":
		return cmdSimple(socket, CmdRemoveCartrige)
	case "removetape":
		return cmdSimple(socket, CmdRemoveTape)
	case "removedisks":
		return cmdSimple(socket, CmdRemoveDisks)
	case "emulator stop", "emulator.stop":
		return cmdSimple(socket, CmdStopEmulator)
	case "emulator restart", "emulator.restart":
		return cmdSimple(socket, CmdRestartEmulator)
	case "emulator debug", "emulator.debug":
		return cmdSimple(socket, CmdBuiltinMonitor)
	case "emulator features", "emulator.features":
		return cmdEmulatorConfig(socket)
	case "bp", "bp ls", "bp.ls":
		return cmdBPList(socket)
	case "bp add", "bp.add":
		return cmdBPAdd(socket, args.BP.Add)
	case "bp del", "bp.del":
		return cmdBPDelete(socket, args.BP.Del)
	case "bp clear", "bp.clear":
		return cmdBPClear(socket)
	case "bp on", "bp.on":
		return cmdBPSetEnabled(socket, true)
	case "bp off", "bp.off":
		return cmdBPSetEnabled(socket, false)
	case "dlist":
		return cmdDumpDList(socket, args.DList)
	case "cpustate":
		return cmdCPUState(socket)
	case "history":
		return cmdHistory(socket, args.History.Count)
	case "status":
		return cmdStatus(socket)
	case "ping":
		return cmdPing(socket)
	case "readmem":
		return cmdReadMem(socket, args.ReadMem)
	case "writemem":
		return cmdWriteMem(socket, args.WriteMem)
	case "disasm":
		return cmdDisasm(socket, args.Disasm)
	case "screen":
		return cmdScreen(socket, args.Screen)
	default:
		return 2
	}
}

func parseCLI(argv []string) (cliArgs, *kong.Context, error) {
	var args cliArgs
	parser, err := kong.New(
		&args,
		kong.Name("go800mon"),
		kong.Description("Atari800 monitor UI and CLI."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:   true,
			FlagsLast: true,
		}),
		kong.Help(colorizedHelpPrinter(kong.DefaultHelpPrinter)),
		kong.ShortHelp(colorizedHelpPrinter(kong.DefaultShortHelpPrinter)),
	)
	if err != nil {
		return args, nil, err
	}
	parsed, err := parser.Parse(argv)
	if err != nil {
		return args, nil, err
	}
	return args, parsed, nil
}

func canFallbackToMonitor(argv []string) bool {
	for i := 0; i < len(argv); i++ {
		token := argv[i]
		switch token {
		case "-s", "--socket":
			if i+1 >= len(argv) || strings.HasPrefix(argv[i+1], "-") {
				return false
			}
			i++
			continue
		case "--":
			return i+1 >= len(argv)
		}
		if strings.HasPrefix(token, "--socket=") || strings.HasPrefix(token, "-s=") {
			continue
		}
		if strings.HasPrefix(token, "-") {
			continue
		}
		return false
	}
	return true
}

func rpcClient(socket string) *RpcClient {
	return NewRpcClient(NewSocketTransport(socket))
}

func cmdMonitor(socket string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := RunMonitor(ctx, socket)
	if err != nil && err != context.Canceled {
		return fail(err)
	}
	return 0
}

func cmdRun(socket string, pathArg string) int {
	path, err := expandPath(pathArg)
	if err != nil {
		return fail(err)
	}
	_, err = rpcClient(socket).Call(context.Background(), CmdRun, []byte(path))
	if err != nil {
		return fail(err)
	}
	return 0
}

func cmdSimple(socket string, cmd Command) int {
	_, err := rpcClient(socket).Call(context.Background(), cmd, nil)
	if err != nil {
		return fail(err)
	}
	return 0
}

func cmdStepLike(socket string, cmd Command) int {
	cl := rpcClient(socket)
	if _, err := cl.Call(context.Background(), cmd, nil); err != nil {
		return fail(err)
	}
	return printCPUState(cl)
}

func cmdDumpDList(socket string, args cliDListCmd) int {
	cl := rpcClient(socket)
	ctx := context.Background()
	var (
		start uint16
		err   error
		dump  []byte
	)
	if args.Address == nil {
		start, err = cl.ReadVector(ctx, DLPTRSAddr)
		if err != nil {
			return fail(err)
		}
		dump, err = cl.ReadDisplayList(ctx)
	} else {
		start, err = memory.ParseHex(*args.Address)
		if err != nil {
			return fail(err)
		}
		dump, err = cl.ReadDisplayListAt(ctx, start)
	}
	if err != nil {
		return fail(err)
	}
	dmactl, err := cl.ReadByte(ctx, DMACTLAddr)
	if err != nil {
		return fail(err)
	}
	if dmactl&0x03 == 0 {
		if hw, hwErr := cl.ReadByte(ctx, DMACTLHWAddr); hwErr == nil {
			dmactl = hw
		}
	}
	dlist := DecodeDisplayList(start, dump)
	for _, c := range dlist.Compacted() {
		if c.Count > 1 {
			fmt.Printf("%04X: %dx %s\n", c.Entry.Addr, c.Count, c.Entry.Description())
		} else {
			fmt.Printf("%04X: %s\n", c.Entry.Addr, c.Entry.Description())
		}
	}
	fmt.Println()
	fmt.Printf("Length: %04X\n", len(dump))
	segs := dlist.ScreenSegments(dmactl)
	if len(segs) > 0 {
		fmt.Println("Screen segments:")
		for i, seg := range segs {
			length := seg.End - seg.Start
			last := (seg.End - 1) & 0xFFFF
			fmt.Printf("#%d %04X-%04X len=%04X antic=%d\n", i+1, seg.Start, last, length, seg.Mode)
		}
	}
	return 0
}

func cmdCPUState(socket string) int {
	return printCPUState(rpcClient(socket))
}

func printCPUState(cl *RpcClient) int {
	cpu, err := cl.CPUState(context.Background())
	if err != nil {
		return fail(err)
	}
	fmt.Println(formatCPU(cpu))
	return 0
}

func cmdHistory(socket string, count int) int {
	entries, err := rpcClient(socket).History(context.Background())
	if err != nil {
		return fail(err)
	}
	if count >= 0 && count < len(entries) {
		entries = entries[:count]
	}
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		ins := disasm.DisasmOne(e.PC, e.OpBytes())
		fmt.Printf("%03d Y=%02X X=%02X PC=%04X  %s\n", (len(entries) - i), e.Y, e.X, e.PC, ins)
	}
	return 0
}

func cmdStatus(socket string) int {
	st, err := rpcClient(socket).Status(context.Background())
	if err != nil {
		return fail(err)
	}
	fmt.Printf(
		"paused=%s crashed=%s emu_ms=%d reset_ms=%d state_seq=%d\n",
		yesNo(st.Paused),
		yesNo(st.Crashed),
		st.EmuMS,
		st.ResetMS,
		st.StateSeq,
	)
	return 0
}

type emulatorCapability struct {
	ID   uint16
	Desc string
}

var emulatorCapabilities = []emulatorCapability{
	{0x0001, "SDL2 video backend (VIDEO_SDL2)"},
	{0x0002, "SDL1 video backend (VIDEO_SDL)"},
	{0x0003, "Sound support (SOUND)"},
	{0x0004, "Callback sound backend (SOUND_CALLBACK)"},
	{0x0005, "Audio recording (AUDIO_RECORDING)"},
	{0x0006, "Video recording (VIDEO_RECORDING)"},
	{0x0007, "Code breakpoints/history (MONITOR_BREAK)"},
	{0x0008, "User breakpoint table (MONITOR_BREAKPOINTS)"},
	{0x0009, "Readline monitor support (MONITOR_READLINE)"},
	{0x000A, "Disassembler label hints (MONITOR_HINTS)"},
	{0x000B, "UTF-8 monitor output (MONITOR_UTF8)"},
	{0x000C, "ANSI monitor output (MONITOR_ANSI)"},
	{0x000D, "Monitor assembler command (MONITOR_ASSEMBLER)"},
	{0x000E, "Monitor profiling/coverage (MONITOR_PROFILE)"},
	{0x000F, "Monitor TRACE command (MONITOR_TRACE)"},
	{0x0010, "NetSIO/FujiNet emulation (NETSIO)"},
	{0x0011, "IDE emulation (IDE)"},
	{0x0012, "R: device support (R_IO_DEVICE)"},
	{0x0013, "Black Box emulation (PBI_BB)"},
	{0x0014, "MIO emulation (PBI_MIO)"},
	{0x0015, "Prototype80 emulation (PBI_PROTO80)"},
	{0x0016, "1400XL/1450XLD emulation (PBI_XLD)"},
	{0x0017, "VoiceBox emulation (VOICEBOX)"},
	{0x0018, "AF80 card emulation (AF80)"},
	{0x0019, "BIT3 card emulation (BIT3)"},
	{0x001A, "XEP80 emulation (XEP80_EMULATION)"},
	{0x001B, "NTSC filter (NTSC_FILTER)"},
	{0x001C, "PAL blending (PAL_BLENDING)"},
	{0x001D, "Crash menu support (CRASH_MENU)"},
	{0x001E, "New cycle-exact core (NEW_CYCLE_EXACT)"},
	{0x001F, "libpng support (HAVE_LIBPNG)"},
	{0x0020, "zlib support (HAVE_LIBZ)"},
}

var bpConditionTypes = map[string]byte{
	"pc":     1,
	"a":      2,
	"x":      3,
	"y":      4,
	"s":      5,
	"read":   6,
	"write":  7,
	"access": 8,
}

var bpTypeNames = map[byte]string{
	1: "pc",
	2: "a",
	3: "x",
	4: "y",
	5: "s",
	6: "read",
	7: "write",
	8: "access",
	9: "mem",
}

var bpOpIDs = map[string]byte{
	"<":  1,
	"<=": 2,
	"=":  3,
	"==": 3,
	"!=": 4,
	">=": 5,
	">":  6,
}

var bpOpNames = map[byte]string{
	1: "<",
	2: "<=",
	3: "==",
	4: "!=",
	5: ">=",
	6: ">",
}

func cmdBPList(socket string) int {
	list, err := rpcClient(socket).BPList(context.Background())
	if err != nil {
		return fail(err)
	}
	fmt.Printf("Enabled: %s\n", formatOnOffBadge(list.Enabled))
	if len(list.Clauses) == 0 {
		fmt.Println("No breakpoint clauses.")
		return 0
	}
	for i, clause := range list.Clauses {
		parts := make([]string, 0, len(clause))
		for _, cond := range clause {
			parts = append(parts, formatBPCondition(cond))
		}
		fmt.Printf("#%02d %s\n", i+1, strings.Join(parts, " && "))
	}
	return 0
}

func cmdBPAdd(socket string, args cliBPAddCmd) int {
	if len(args.Conditions) == 0 {
		return fail(errors.New("Specify at least one condition."))
	}
	conds := make([]BreakpointCondition, 0, len(args.Conditions))
	for _, expr := range args.Conditions {
		cond, err := parseBPCondition(expr)
		if err != nil {
			return fail(err)
		}
		conds = append(conds, cond)
	}
	idx, err := rpcClient(socket).BPAddClause(context.Background(), conds)
	if err != nil {
		return fail(err)
	}
	fmt.Printf("Added clause #%d\n", int(idx)+1)
	return 0
}

func cmdBPDelete(socket string, args cliBPDeleteCmd) int {
	if args.Index <= 0 {
		return fail(errors.New("Clause index must be >= 1."))
	}
	if err := rpcClient(socket).BPDeleteClause(context.Background(), uint16(args.Index-1)); err != nil {
		return fail(err)
	}
	return cmdBPList(socket)
}

func cmdBPClear(socket string) int {
	if err := rpcClient(socket).BPClear(context.Background()); err != nil {
		return fail(err)
	}
	return cmdBPList(socket)
}

func cmdBPSetEnabled(socket string, enabled bool) int {
	if _, err := rpcClient(socket).BPSetEnabled(context.Background(), enabled); err != nil {
		return fail(err)
	}
	return cmdBPList(socket)
}

func splitBPExpression(expr string) (string, string, string, bool) {
	text := strings.TrimSpace(expr)
	for _, op := range []string{"<=", ">=", "==", "!=", "=", "<", ">"} {
		pos := strings.Index(text, op)
		if pos <= 0 {
			continue
		}
		left := strings.TrimSpace(text[:pos])
		right := strings.TrimSpace(text[pos+len(op):])
		if right == "" {
			break
		}
		return left, op, right, true
	}
	return "", "", "", false
}

func parseBPCondition(expr string) (BreakpointCondition, error) {
	left, opText, valueText, ok := splitBPExpression(expr)
	if !ok {
		return BreakpointCondition{}, fmt.Errorf("Invalid breakpoint condition: %s", expr)
	}
	op, ok := bpOpIDs[opText]
	if !ok {
		return BreakpointCondition{}, fmt.Errorf("Invalid breakpoint operator in condition: %s", expr)
	}
	leftKey := strings.ToLower(strings.TrimSpace(left))
	cond := BreakpointCondition{
		Op:   op,
		Addr: 0,
	}
	if t, ok := bpConditionTypes[leftKey]; ok {
		cond.Type = t
	} else if strings.HasPrefix(leftKey, "mem[") && strings.HasSuffix(leftKey, "]") {
		addr, err := memory.ParseHex(leftKey[4 : len(leftKey)-1])
		if err != nil {
			return BreakpointCondition{}, fmt.Errorf("Invalid memory address in condition: %s", expr)
		}
		cond.Type = 9
		cond.Addr = addr
	} else if strings.HasPrefix(leftKey, "mem:") {
		addr, err := memory.ParseHex(leftKey[4:])
		if err != nil {
			return BreakpointCondition{}, fmt.Errorf("Invalid memory address in condition: %s", expr)
		}
		cond.Type = 9
		cond.Addr = addr
	} else {
		return BreakpointCondition{}, fmt.Errorf("Invalid breakpoint source in condition: %s", expr)
	}
	value, err := memory.ParseHex(valueText)
	if err != nil {
		return BreakpointCondition{}, fmt.Errorf("Invalid breakpoint value in condition: %s", expr)
	}
	cond.Value = value
	return cond, nil
}

func formatBPValue(condType byte, value uint16) string {
	if condType == 2 || condType == 3 || condType == 4 || condType == 5 {
		return fmt.Sprintf("$%02X", value)
	}
	return fmt.Sprintf("$%04X", value)
}

func formatBPCondition(cond BreakpointCondition) string {
	op := bpOpNames[cond.Op]
	if op == "" {
		op = fmt.Sprintf("op%d", cond.Op)
	}
	if cond.Type == 9 {
		return fmt.Sprintf("mem[%04X] %s %s", cond.Addr, op, formatBPValue(cond.Type, cond.Value))
	}
	name := bpTypeNames[cond.Type]
	if name == "" {
		name = fmt.Sprintf("type%d", cond.Type)
	}
	return fmt.Sprintf("%s %s %s", name, op, formatBPValue(cond.Type, cond.Value))
}

func cmdEmulatorConfig(socket string) int {
	caps, err := rpcClient(socket).Config(context.Background())
	if err != nil {
		return fail(err)
	}
	enabled := map[uint16]bool{}
	for _, id := range caps {
		enabled[id] = true
	}
	known := map[uint16]bool{}
	for _, cap := range emulatorCapabilities {
		known[cap.ID] = true
		fmt.Printf("%s %s\n", formatOnOffBadge(enabled[cap.ID]), cap.Desc)
	}
	for _, id := range caps {
		if known[id] {
			continue
		}
		fmt.Printf("%s Unknown capability 0x%04X\n", formatOnOffBadge(true), id)
	}
	return 0
}

func cmdPing(socket string) int {
	data, err := rpcClient(socket).Call(context.Background(), CmdPing, nil)
	if err != nil {
		return fail(err)
	}
	if len(data) > 0 {
		_, _ = os.Stdout.Write(data)
	}
	return 0
}

func cmdReadMem(socket string, args cliReadMemCmd) int {
	addr, err := memory.ParseHex(args.Addr)
	if err != nil {
		return fail(err)
	}
	length, err := memory.ParseHex(args.Length)
	if err != nil {
		return fail(err)
	}
	data, err := rpcClient(socket).ReadMemoryChunked(context.Background(), addr, int(length))
	if err != nil {
		return fail(err)
	}
	cols := 0
	columnsProvided := args.Columns != nil
	if columnsProvided {
		cols = *args.Columns
	}
	return dumpMemory(
		addr,
		int(length),
		data,
		args.Raw,
		args.JSON,
		args.ATASCII,
		cols,
		columnsProvided,
		!args.NoHex,
		!args.NoASCII,
	)
}

func cmdWriteMem(socket string, args cliWriteMemCmd) int {
	addr, err := memory.ParseHex(args.Addr)
	if err != nil {
		return fail(err)
	}
	hasBytes := len(args.Bytes) > 0
	hasHex := args.Hex != nil
	hasText := args.Text != nil
	if btoi(hasBytes)+btoi(hasHex)+btoi(hasText) != 1 {
		return fail(errors.New("Specify exactly one payload: <bytes...>, --hex, or --text."))
	}
	if args.ATASCII && !hasText {
		return fail(errors.New("--atascii is only valid with --text."))
	}
	data, err := resolveWriteMemData(args, hasBytes, hasHex)
	if err != nil {
		return fail(err)
	}
	if len(data) == 0 {
		return fail(errors.New("No data to write."))
	}
	if len(data) > 0xFFFF {
		return fail(fmt.Errorf("Data too long: %d bytes (max 65535).", len(data)))
	}
	if args.Screen {
		data = toScreenCodes(data)
	}
	if err := rpcClient(socket).WriteMemory(context.Background(), addr, data); err != nil {
		return fail(err)
	}
	return 0
}

func resolveWriteMemData(args cliWriteMemCmd, hasBytes bool, hasHex bool) ([]byte, error) {
	if hasBytes {
		return parseHexValues(args.Bytes)
	}
	if hasHex {
		text := strings.TrimSpace(*args.Hex)
		if text == "-" {
			raw, err := io.ReadAll(os.Stdin)
			if err != nil {
				return nil, err
			}
			text = string(raw)
		}
		return parseHexPayload(text)
	}
	text := *args.Text
	if text == "-" {
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
		text = string(raw)
	}
	if args.ATASCII {
		return EncodeATASCIIText(text)
	}
	return []byte(text), nil
}

func parseHexValues(tokens []string) ([]byte, error) {
	out := make([]byte, 0, len(tokens))
	for _, tok := range tokens {
		v, err := memory.ParseHex(tok)
		if err != nil {
			return nil, fmt.Errorf("Invalid hex value: %s", tok)
		}
		if v <= 0xFF {
			out = append(out, byte(v))
			continue
		}
		out = append(out, byte(v&0xFF), byte(v>>8))
	}
	return out, nil
}

func parseHexByteToken(token string) (byte, error) {
	v, err := memory.ParseHex(token)
	if err != nil {
		return 0, fmt.Errorf("Invalid hex byte: %s", token)
	}
	if v > 0xFF {
		return 0, fmt.Errorf("Hex byte out of range: %s", token)
	}
	return byte(v), nil
}

func parseHexPayload(text string) ([]byte, error) {
	normalized := strings.ReplaceAll(text, ",", " ")
	fields := strings.Fields(normalized)
	if len(fields) == 0 {
		return nil, errors.New("Hex payload is empty.")
	}
	if len(fields) > 1 {
		out := make([]byte, 0, len(fields))
		for _, tok := range fields {
			b, err := parseHexByteToken(tok)
			if err != nil {
				return nil, err
			}
			out = append(out, b)
		}
		return out, nil
	}
	value := strings.TrimSpace(strings.ToLower(fields[0]))
	value = strings.TrimPrefix(value, "$")
	value = strings.TrimPrefix(value, "0x")
	if value == "" {
		return nil, errors.New("Hex payload is empty.")
	}
	if len(value)%2 != 0 {
		return nil, errors.New("Hex payload must have an even number of digits.")
	}
	out := make([]byte, 0, len(value)/2)
	for i := 0; i < len(value); i += 2 {
		v, err := strconv.ParseUint(value[i:i+2], 16, 8)
		if err != nil {
			return nil, errors.New("Invalid hex payload.")
		}
		out = append(out, byte(v))
	}
	return out, nil
}

func toScreenCodes(data []byte) []byte {
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = ATASCIIToScreen(b)
	}
	return out
}

func cmdDisasm(socket string, args cliDisasmCmd) int {
	addr, err := memory.ParseHex(args.Addr)
	if err != nil {
		return fail(err)
	}
	length, err := memory.ParseHex(args.Length)
	if err != nil {
		return fail(err)
	}
	data, err := rpcClient(socket).ReadMemoryChunked(context.Background(), addr, int(length))
	if err != nil {
		return fail(err)
	}
	for _, line := range disasm.Disasm(addr, data) {
		fmt.Println(line)
	}
	return 0
}

func btoi(v bool) int {
	if v {
		return 1
	}
	return 0
}

func cmdScreen(socket string, args cliScreenCmd) int {
	if args.List && args.Segment != nil {
		fmt.Fprintln(os.Stderr, "--list cannot be used with a segment number")
		return 1
	}
	cl := rpcClient(socket)
	ctx := context.Background()
	start, err := cl.ReadVector(ctx, DLPTRSAddr)
	if err != nil {
		return fail(err)
	}
	dump, err := cl.ReadDisplayList(ctx)
	if err != nil {
		return fail(err)
	}
	dmactl, err := cl.ReadByte(ctx, DMACTLAddr)
	if err != nil {
		return fail(err)
	}
	if dmactl&0x03 == 0 {
		if hw, hwErr := cl.ReadByte(ctx, DMACTLHWAddr); hwErr == nil {
			dmactl = hw
		}
	}
	dlist := dl.Decode(start, dump)
	segments := dlist.ScreenSegments(dmactl)
	if len(segments) == 0 {
		fmt.Fprintln(os.Stderr, "No screen segments found.")
		return 1
	}
	if args.List {
		for i, seg := range segments {
			length := seg.End - seg.Start
			last := (seg.End - 1) & 0xFFFF
			fmt.Printf("#%d %04X-%04X len=%04X antic=%d\n", i+1, seg.Start, last, length, seg.Mode)
		}
		return 0
	}
	mapper := dl.NewMemoryMapper(dlist, dmactl, 4096)
	if args.Segment == nil {
		if args.Columns == nil && !args.Raw && !args.JSON {
			rows := make([]memory.DumpRow, 0)
			for _, row := range mapper.RowRanges() {
				if row.Addr == nil || row.Length <= 0 {
					continue
				}
				chunk, err := cl.ReadMemory(ctx, *row.Addr, uint16(row.Length))
				if err != nil {
					return fail(err)
				}
				if len(chunk) == 0 {
					continue
				}
				rowCopy := make([]byte, len(chunk))
				copy(rowCopy, chunk)
				rows = append(rows, memory.DumpRow{
					Address: *row.Addr,
					Data:    rowCopy,
				})
			}
			if len(rows) > 0 {
				fmt.Println(memory.DumpHumanRows(rows, args.ATASCII, !args.NoHex, !args.NoASCII))
				return 0
			}
		}
		data := make([]byte, 0)
		for _, seg := range segments {
			chunk, err := cl.ReadMemoryChunked(ctx, uint16(seg.Start&0xFFFF), seg.End-seg.Start)
			if err != nil {
				return fail(err)
			}
			data = append(data, chunk...)
		}
		cols := 0
		columnsProvided := args.Columns != nil
		if columnsProvided {
			cols = *args.Columns
		}
		return dumpMemory(
			uint16(segments[0].Start&0xFFFF),
			len(data),
			data,
			args.Raw,
			args.JSON,
			args.ATASCII,
			cols,
			columnsProvided,
			!args.NoHex,
			!args.NoASCII,
		)
	}
	idx := *args.Segment - 1
	if idx < 0 || idx >= len(segments) {
		fmt.Fprintf(os.Stderr, "Segment out of range (1-%d)\n", len(segments))
		return 1
	}
	seg := segments[idx]
	length := seg.End - seg.Start
	data, err := cl.ReadMemoryChunked(ctx, uint16(seg.Start&0xFFFF), length)
	if err != nil {
		return fail(err)
	}
	if args.Columns == nil && !args.Raw && !args.JSON {
		rows := make([]memory.DumpRow, 0)
		for _, row := range mapper.RowRanges() {
			if row.Addr == nil || row.Length <= 0 {
				continue
			}
			addr := int(*row.Addr)
			if addr < seg.Start || addr >= seg.End {
				continue
			}
			rel := addr - seg.Start
			if rel < 0 || rel >= len(data) {
				continue
			}
			rowEnd := rel + row.Length
			if rowEnd > len(data) {
				rowEnd = len(data)
			}
			chunk := data[rel:rowEnd]
			if len(chunk) == 0 {
				continue
			}
			rowCopy := make([]byte, len(chunk))
			copy(rowCopy, chunk)
			rows = append(rows, memory.DumpRow{
				Address: uint16(addr & 0xFFFF),
				Data:    rowCopy,
			})
		}
		if len(rows) > 0 {
			fmt.Println(memory.DumpHumanRows(rows, args.ATASCII, !args.NoHex, !args.NoASCII))
			return 0
		}
	}
	cols := 0
	columnsProvided := args.Columns != nil
	if columnsProvided {
		cols = *args.Columns
	}
	if cols == 0 {
		if c := mapper.BytesPerLine(seg.Mode); c > 0 {
			cols = c
		}
	}
	return dumpMemory(
		uint16(seg.Start&0xFFFF),
		length,
		data,
		args.Raw,
		args.JSON,
		args.ATASCII,
		cols,
		columnsProvided,
		!args.NoHex,
		!args.NoASCII,
	)
}

func dumpMemory(address uint16, length int, data []byte, raw bool, asJSON bool, useATASCII bool, columns int, columnsProvided bool, showHex bool, showASCII bool) int {
	if columnsProvided && (raw || asJSON) {
		fmt.Fprintln(os.Stderr, "--columns is only valid for formatted output")
		return 1
	}
	if raw {
		out := memory.DumpRaw(data, useATASCII)
		if len(out) > 0 {
			_, _ = os.Stdout.Write(out)
		}
		return 0
	}
	if asJSON {
		text, err := memory.DumpJSON(address, data, useATASCII)
		if err != nil {
			return fail(err)
		}
		fmt.Println(text)
		return 0
	}
	fmt.Println(memory.DumpHuman(address, length, data, useATASCII, columns, showHex, showASCII))
	return 0
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return real, nil
	}
	return abs, nil
}

func colorizedHelpPrinter(base kong.HelpPrinter) kong.HelpPrinter {
	return func(options kong.HelpOptions, ctx *kong.Context) error {
		out := ctx.Stdout
		var buf bytes.Buffer
		ctx.Stdout = &buf
		err := base(options, ctx)
		ctx.Stdout = out
		if err != nil {
			return err
		}
		text := buf.String()
		if !helpColorEnabled() {
			_, werr := io.WriteString(out, text)
			return werr
		}
		_, werr := io.WriteString(out, colorizeHelpText(text))
		return werr
	}
}

func helpColorEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("A800MON_HELP_COLOR"))) {
	case "always":
		return true
	case "never":
		return false
	}
	term := os.Getenv("TERM")
	return term != "" && term != "dumb"
}

func colorizeHelpText(text string) string {
	const (
		reset = "\x1b[0m"
		head  = "\x1b[1;36m"
		cmd   = "\x1b[1;33m"
		flag  = "\x1b[32m"
		dim   = "\x1b[2m"
	)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		leading := len(line) - len(strings.TrimLeft(line, " "))
		if strings.HasPrefix(trim, "Usage:") ||
			trim == "Commands:" ||
			trim == "Arguments:" ||
			trim == "Flags:" {
			lines[i] = head + trim + reset
			continue
		}
		if strings.HasPrefix(trim, "Run \"") {
			lines[i] = dim + line + reset
			continue
		}
		if leading <= 6 && strings.HasPrefix(trim, "-") {
			lines[i] = colorizeHelpLeadingToken(line, flag, reset)
			continue
		}
		if leading == 2 && trim != "" && !strings.HasPrefix(trim, "-") &&
			(strings.Contains(trim, "  ") || strings.Contains(trim, "(")) {
			lines[i] = colorizeHelpLeadingToken(line, cmd, reset)
		}
	}
	return strings.Join(lines, "\n")
}

func colorizeHelpLeadingToken(line string, color string, reset string) string {
	indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
	trim := strings.TrimSpace(line)
	sep := strings.Index(trim, "  ")
	if sep < 0 {
		return indent + color + trim + reset
	}
	return indent + color + trim[:sep] + reset + trim[sep:]
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, formatCliError(err))
	return 1
}

func formatCliError(err error) string {
	if err == nil {
		return ""
	}
	var commandErr CommandError
	if errors.As(err, &commandErr) {
		msg := strings.TrimSpace(string(commandErr.Data))
		if msg == "" {
			msg = err.Error()
		}
		return formatCliBadge(fmt.Sprintf("%d", commandErr.Status), msg)
	}
	return formatCliBadge("ERR", err.Error())
}

func formatCliBadge(code string, msg string) string {
	badge := " " + code + " "
	if cliColorEnabled() {
		return "\x1b[41;97;1m" + badge + "\x1b[0m " + msg
	}
	return "[" + code + "] " + msg
}

func cliColorEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("A800MON_COLOR"))) {
	case "always":
		return true
	case "never":
		return false
	}
	return helpColorEnabled()
}

func formatOnOffBadge(enabled bool) string {
	text := "OFF"
	if enabled {
		text = "ON "
	}
	badge := " " + text + " "
	if !cliColorEnabled() {
		return badge
	}
	if enabled {
		return "\x1b[42;30m" + badge + "\x1b[0m"
	}
	return "\x1b[41;97;1m" + badge + "\x1b[0m"
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
