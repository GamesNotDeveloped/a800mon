package a800mon

var debugLog []string

func DebugLog(text string) {
	debugLog = append(debugLog, text)
}

func DebugDump() []string {
	out := make([]string, len(debugLog))
	copy(out, debugLog)
	return out
}
