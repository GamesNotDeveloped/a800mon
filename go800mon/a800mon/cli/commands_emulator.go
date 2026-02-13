package cli

import (
	"context"
	"fmt"
)

func cmdEmulatorReboot(socket string, args cliEmulatorRebootCmd) int {
	command := CmdWarmstart
	if args.Cold {
		command = CmdColdstart
	}
	return cmdSimple(socket, command)
}

func cmdStatus(socket string) int {
	st, err := rpcClient(socket).Status(context.Background())
	if err != nil {
		return fail(err)
	}
	fmt.Printf(
		"paused=%t crashed=%t machine_type=%s emu_ms=%d reset_ms=%d state_seq=%d\n",
		st.Paused,
		st.Crashed,
		StatusMachineName(st.MachineType),
		st.EmuMS,
		st.ResetMS,
		st.StateSeq,
	)
	return 0
}

func cmdSysinfo(socket string) int {
	info, err := rpcClient(socket).Sysinfo(context.Background())
	if err != nil {
		return fail(err)
	}
	fmt.Printf(
		"basic_enabled=%t tv_pal=%t machine_family=%s os_revision=%s "+
			"basic_revision=%s builtin_game_revision=%s\n",
		info.BasicEnabled,
		info.TVPAL,
		StatusMachineFamilyName(info.MachineFamily),
		StatusOSRevisionName(info.OSRevision),
		StatusBasicRevisionName(info.BasicRevision),
		StatusBuiltinGameRevisionName(info.BuiltinGameRevision),
	)
	return 0
}

func cmdEmulatorConfig(socket string) int {
	caps, err := rpcClient(socket).BuildFeatures(context.Background())
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
