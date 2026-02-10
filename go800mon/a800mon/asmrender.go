package a800mon

const asmCommentCol = 18

func printAsmRow(window *Window, row DisasmRow, revAttr int) {
	if row.Mnemonic == "" {
		window.Print(row.AsmText, revAttr, false)
		return
	}
	coreLen := len([]rune(row.Mnemonic))
	window.Print(row.Mnemonic, ColorMnemonic.Attr()|revAttr, false)
	if row.Operand != "" {
		window.Print(" ", revAttr, false)
		coreLen += 1 + len([]rune(row.Operand))
		if row.FlowTarget == nil || !row.HasOperandAddr {
			window.Print(row.Operand, revAttr, false)
		} else {
			start := row.OperandAddrPos[0]
			end := row.OperandAddrPos[1]
			r := []rune(row.Operand)
			if start < 0 {
				start = 0
			}
			if end > len(r) {
				end = len(r)
			}
			if start > end {
				start = end
			}
			window.Print(string(r[:start]), revAttr, false)
			window.Print(string(r[start:end]), ColorAddress.Attr()|revAttr, false)
			window.Print(string(r[end:]), revAttr, false)
		}
	}
	if row.Comment == "" {
		return
	}
	if coreLen < asmCommentCol {
		window.Print(spaces(asmCommentCol-coreLen), revAttr, false)
	}
	window.Print(" ", revAttr, false)
	window.Print(row.Comment, ColorComment.Attr()|revAttr, false)
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]rune, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
