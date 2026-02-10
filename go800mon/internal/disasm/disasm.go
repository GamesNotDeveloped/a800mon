package disasm

import "fmt"

import "go800mon/internal/memorymap"

type DecodedInstruction struct {
	Addr           uint16
	Size           int
	Raw            []byte
	RawText        string
	Mnemonic       string
	Operand        string
	Comment        string
	AsmText        string
	Addressing     string
	FlowTarget     *uint16
	OperandAddrPos *[2]int
}

var flowMnemonics = map[string]struct{}{
	"JMP": {}, "JSR": {}, "BCC": {}, "BCS": {}, "BEQ": {}, "BMI": {},
	"BNE": {}, "BPL": {}, "BVC": {}, "BVS": {}, "BRA": {},
}

func Disasm(startAddr uint16, data []byte) []string {
	decoded := Decode(startAddr, data)
	out := make([]string, 0, len(decoded))
	for _, ins := range decoded {
		out = append(out, fmt.Sprintf("%04X: %-8s %s", ins.Addr, ins.RawText, ins.AsmText))
	}
	return out
}

func DisasmOne(startAddr uint16, data []byte) string {
	ins := DecodeOne(startAddr, data)
	if ins == nil {
		return ""
	}
	return fmt.Sprintf("%-8s %s", ins.RawText, ins.AsmText)
}

func DecodeOne(startAddr uint16, data []byte) *DecodedInstruction {
	decoded := Decode(startAddr, data)
	if len(decoded) == 0 {
		return nil
	}
	return &decoded[0]
}

func Decode(startAddr uint16, data []byte) []DecodedInstruction {
	if len(data) == 0 {
		return nil
	}
	pc := startAddr
	consumed := 0
	out := make([]DecodedInstruction, 0, len(data)/2)
	for consumed < len(data) {
		op := data[consumed]
		mn := opMnemonic[op]
		mode := opMode[op]
		size := modeSize(mode)
		if size < 1 {
			size = 1
		}
		remain := len(data) - consumed
		if size > remain {
			size = remain
		}
		raw := make([]byte, size)
		copy(raw, data[consumed:consumed+size])
		rawText := fmtBytes(raw)
		operand, target, span := formatOperand(mode, pc, raw)
		if mn == "???" {
			mn = ".DB"
			operand = fmt.Sprintf("$%02X", op)
			target = nil
			span = nil
		}
		baseAsm := mn
		if operand != "" {
			baseAsm += " " + operand
		}
		comment := ""
		if target != nil {
			if symbol := memorymap.Lookup(*target); symbol != "" {
				comment = ";" + symbol
			}
		}
		asmText := baseAsm
		if comment != "" {
			if len(baseAsm) < 18 {
				asmText = baseAsm + spaces(18-len(baseAsm))
			}
			asmText += " " + comment
		}
		var flowTarget *uint16
		if _, ok := flowMnemonics[mn]; ok && target != nil {
			v := *target
			flowTarget = &v
		}
		out = append(out, DecodedInstruction{
			Addr:           pc,
			Size:           size,
			Raw:            raw,
			RawText:        rawText,
			Mnemonic:       mn,
			Operand:        operand,
			Comment:        comment,
			AsmText:        asmText,
			Addressing:     mode,
			FlowTarget:     flowTarget,
			OperandAddrPos: span,
		})
		consumed += size
		pc = uint16((int(pc) + size) & 0xFFFF)
	}
	return out
}

func modeSize(mode string) int {
	switch mode {
	case "imp", "acc":
		return 1
	case "imm", "inx", "iny", "rel", "zpg", "zpx", "zpy":
		return 2
	case "abs", "abx", "aby", "ind":
		return 3
	default:
		return 1
	}
}

func formatOperand(mode string, pc uint16, raw []byte) (string, *uint16, *[2]int) {
	if len(raw) == 0 {
		return "", nil, nil
	}
	byteAt := func(i int) byte {
		if i < 0 || i >= len(raw) {
			return 0
		}
		return raw[i]
	}
	wordAt := func(i int) uint16 {
		lo := uint16(byteAt(i))
		hi := uint16(byteAt(i + 1))
		return lo | (hi << 8)
	}
	mkSpan := func(start, end int) *[2]int {
		v := [2]int{start, end}
		return &v
	}
	switch mode {
	case "acc":
		return "A", nil, nil
	case "abs":
		addr := wordAt(1)
		t := fmt.Sprintf("$%04X", addr)
		return t, &addr, mkSpan(0, len(t))
	case "abx":
		addr := wordAt(1)
		t := fmt.Sprintf("$%04X", addr)
		return t + ",X", &addr, mkSpan(0, len(t))
	case "aby":
		addr := wordAt(1)
		t := fmt.Sprintf("$%04X", addr)
		return t + ",Y", &addr, mkSpan(0, len(t))
	case "imm":
		return fmt.Sprintf("#$%02X", byteAt(1)), nil, nil
	case "imp":
		return "", nil, nil
	case "ind":
		addr := wordAt(1)
		t := fmt.Sprintf("$%04X", addr)
		return "(" + t + ")", &addr, mkSpan(1, 1+len(t))
	case "iny":
		zp := uint16(byteAt(1))
		t := fmt.Sprintf("$%02X", byteAt(1))
		return "(" + t + "),Y", &zp, mkSpan(1, 1+len(t))
	case "inx":
		zp := uint16(byteAt(1))
		t := fmt.Sprintf("$%02X", byteAt(1))
		return "(" + t + ",X)", &zp, mkSpan(1, 1+len(t))
	case "rel":
		off := int8(byteAt(1))
		target := uint16((int(pc) + 2 + int(off)) & 0xFFFF)
		t := fmt.Sprintf("$%04X", target)
		return t, &target, mkSpan(0, len(t))
	case "zpg":
		zp := uint16(byteAt(1))
		t := fmt.Sprintf("$%02X", byteAt(1))
		return t, &zp, mkSpan(0, len(t))
	case "zpx":
		zp := uint16(byteAt(1))
		t := fmt.Sprintf("$%02X", byteAt(1))
		return t + ",X", &zp, mkSpan(0, len(t))
	case "zpy":
		zp := uint16(byteAt(1))
		t := fmt.Sprintf("$%02X", byteAt(1))
		return t + ",Y", &zp, mkSpan(0, len(t))
	default:
		return "", nil, nil
	}
}

func fmtBytes(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	out := fmt.Sprintf("%02X", raw[0])
	for i := 1; i < len(raw); i++ {
		out += fmt.Sprintf(" %02X", raw[i])
	}
	return out
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	out := ""
	for i := 0; i < n; i++ {
		out += " "
	}
	return out
}
