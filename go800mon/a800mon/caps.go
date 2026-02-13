package a800mon

const (
	CapMonitorBreak       uint16 = 0x0001
	CapMonitorBreakpoints uint16 = 0x0002
	CapEmulationAtari800  uint16 = 0x0003
	CapEmulationXEXL      uint16 = 0x0004
	CapEmulationAtari5200 uint16 = 0x0005

	StatusMachineAtari800  byte = 0
	StatusMachineXLXE      byte = 1
	StatusMachineAtari5200 byte = 2
)

func StatusMachineName(machineType byte) string {
	switch machineType {
	case StatusMachineAtari800:
		return "atari800"
	case StatusMachineXLXE:
		return "xl_xe"
	case StatusMachineAtari5200:
		return "atari5200"
	default:
		return "unknown"
	}
}
