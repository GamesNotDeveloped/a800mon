package memorymap

import "strings"

func FindByComment(query string) (uint16, bool) {
	q := strings.TrimSpace(query)
	q = strings.TrimPrefix(q, ";")
	q = strings.TrimSpace(q)
	if q == "" {
		return 0, false
	}
	q = strings.ToLower(q)

	var exactAddr uint16
	exactOK := false
	var prefixAddr uint16
	prefixOK := false
	var containsAddr uint16
	containsOK := false

	for addr, name := range symbols {
		s := strings.ToLower(name)
		if s == q {
			if !exactOK || addr < exactAddr {
				exactAddr = addr
				exactOK = true
			}
			continue
		}
		if strings.HasPrefix(s, q) {
			if !prefixOK || addr < prefixAddr {
				prefixAddr = addr
				prefixOK = true
			}
			continue
		}
		if strings.Contains(s, q) {
			if !containsOK || addr < containsAddr {
				containsAddr = addr
				containsOK = true
			}
		}
	}

	if exactOK {
		return exactAddr, true
	}
	if prefixOK {
		return prefixAddr, true
	}
	if containsOK {
		return containsAddr, true
	}
	return 0, false
}
