package a800mon

var statusMachineFamilyNames = map[byte]string{
	0: "atari800",
	1: "xl_xe",
	2: "atari5200",
}

func StatusMachineFamilyName(machineFamily byte) string {
	if name, ok := statusMachineFamilyNames[machineFamily]; ok {
		return name
	}
	return "unknown"
}

var statusOSRevisionNames = map[byte]string{
	0x00: "800_a_ntsc",
	0x01: "800_a_pal",
	0x02: "800_b_ntsc",
	0x03: "800_custom",
	0x04: "800_altirra",
	0x20: "xl_10",
	0x21: "xl_11",
	0x22: "xl_1",
	0x23: "xl_2",
	0x24: "xl_3a",
	0x25: "xl_3b",
	0x26: "xl_5",
	0x27: "xl_3",
	0x28: "xl_4",
	0x29: "xl_59",
	0x2A: "xl_59a",
	0x2B: "xl_custom",
	0x2C: "xl_altirra",
	0x40: "5200_orig",
	0x41: "5200_a",
	0x42: "5200_custom",
	0x43: "5200_altirra",
	0xFF: "none",
}

func StatusOSRevisionName(revision byte) string {
	if name, ok := statusOSRevisionNames[revision]; ok {
		return name
	}
	return "unknown"
}

var statusBasicRevisionNames = map[byte]string{
	0x00: "a",
	0x01: "b",
	0x02: "c",
	0x03: "custom",
	0x04: "altirra",
	0xFF: "none",
}

func StatusBasicRevisionName(revision byte) string {
	if name, ok := statusBasicRevisionNames[revision]; ok {
		return name
	}
	return "unknown"
}

var statusBuiltinGameRevisionNames = map[byte]string{
	0x00: "orig",
	0x01: "custom",
	0xFF: "none",
}

func StatusBuiltinGameRevisionName(revision byte) string {
	if name, ok := statusBuiltinGameRevisionNames[revision]; ok {
		return name
	}
	return "unknown"
}
