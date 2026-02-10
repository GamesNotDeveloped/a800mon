package atascii

import "fmt"

var table = []string{
	"♥", "┣", "┃", "┛", "┫", "┓", "╱", "╲", "◢", "◗", "◣", "◝", "◘", "◔", "▁", "◖",
	"♣", "┏", "━", "╋", "⬤", "▄", "▎", "┳", "┻", "▌", "┗", "␛", "↑", "↓", "←", "→",
	" ", "!", "\"", "#", "$", "%", "&", "'", "(", ")", "*", "+", ",", "-", ".", "/",
	"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", ":", ";", "<", "=", ">", "?",
	"@", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O",
	"P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z", "[", "\\", "]", "^", "_",
	"◆", "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o",
	"p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z", "♠", "|", "↰", "◀", "▶",
}

var encodeMap = func() map[string]byte {
	m := make(map[string]byte, len(table))
	for i, ch := range table {
		if _, ok := m[ch]; !ok {
			m[ch] = byte(i)
		}
	}
	return m
}()

func ScreenToATASCII(b byte) byte {
	c := b & 0x7F
	if c < 64 {
		c += 32
	} else if c < 96 {
		c -= 64
	}
	return c | (b & 0x80)
}

func ATASCIIToScreen(b byte) byte {
	c := b & 0x7F
	if c < 32 {
		c += 64
	} else if c < 96 {
		c -= 32
	}
	return c | (b & 0x80)
}

func LookupPrintable(b byte) string {
	return table[b&0x7F]
}

func EncodeText(text string) ([]byte, error) {
	out := make([]byte, 0, len(text))
	for _, r := range text {
		b, ok := encodeMap[string(r)]
		if !ok {
			return nil, fmt.Errorf("cannot encode character to ATASCII: %q", string(r))
		}
		out = append(out, b)
	}
	return out, nil
}
