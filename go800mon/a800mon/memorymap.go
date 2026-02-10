package a800mon

import imap "go800mon/internal/memorymap"

func LookupSymbol(addr uint16) string {
	return imap.Lookup(addr)
}

func FindSymbolByComment(query string) (uint16, bool) {
	return imap.FindByComment(query)
}
