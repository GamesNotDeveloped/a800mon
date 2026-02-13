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
	StatusMachineAtari1200XL          byte = 3
	StatusMachineAtari800XL           byte = 4
	StatusMachineAtari130XE           byte = 5
	StatusMachineAtari320XECompyShop  byte = 6
	StatusMachineAtari320XERambo      byte = 7
	StatusMachineAtari576XE           byte = 8
	StatusMachineAtari1088XE          byte = 9
	StatusMachineAtariXEGS            byte = 10
	StatusMachineAtari400             byte = 11
	StatusMachineAtari600XL           byte = 12
)

func StatusMachineName(machineType byte) string {
	switch machineType {
	case StatusMachineAtari800:
		return "atari800"
	case StatusMachineXLXE:
		return "xl_xe"
	case StatusMachineAtari5200:
		return "atari5200"
	case StatusMachineAtari1200XL:
		return "atari1200xl"
	case StatusMachineAtari800XL:
		return "atari800xl"
	case StatusMachineAtari130XE:
		return "atari130xe"
	case StatusMachineAtari320XECompyShop:
		return "atari320xe_compy_shop"
	case StatusMachineAtari320XERambo:
		return "atari320xe_rambo"
	case StatusMachineAtari576XE:
		return "atari576xe"
	case StatusMachineAtari1088XE:
		return "atari1088xe"
	case StatusMachineAtariXEGS:
		return "atarixegs"
	case StatusMachineAtari400:
		return "atari400"
	case StatusMachineAtari600XL:
		return "atari600xl"
	default:
		return "unknown"
	}
}
