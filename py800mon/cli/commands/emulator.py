import sys

from ...emulator import (
    EMULATOR_CAPABILITIES,
    STATUS_BASIC_REVISION_NAMES,
    STATUS_BUILTIN_GAME_REVISION_NAMES,
    STATUS_MACHINE_FAMILY_NAMES,
    STATUS_MACHINE_NAMES,
    STATUS_OS_REVISION_NAMES,
    format_status_name,
)
from ...rpc import Command
from ..common import (
    async_to_sync,
    rpc_client,
)
from ..utils import format_capability_lines


def register(subparsers):
    emulator = subparsers.add_parser(
        "emulator",
        aliases=["e"],
        help="Emulator control commands.",
    )
    emulator_sub = emulator.add_subparsers(dest="emulator_cmd", metavar="subcmd")

    reboot = emulator_sub.add_parser("reboot", help="Reboot emulation.")
    mode = reboot.add_mutually_exclusive_group()
    mode.add_argument("--cold", action="store_true", help="Cold start.")
    mode.add_argument("--warm", action="store_true", help="Warm start (default).")
    reboot.set_defaults(func=cmd_reboot)

    status = emulator_sub.add_parser("status", help="Get status.")
    status.set_defaults(func=cmd_status)

    sysinfo = emulator_sub.add_parser("sysinfo", help="Get system info.")
    sysinfo.set_defaults(func=cmd_sysinfo)

    stop = emulator_sub.add_parser("stop", help="Stop emulator.")
    stop.set_defaults(func=cmd_stop)

    restart = emulator_sub.add_parser("restart", help="Restart emulator.")
    restart.set_defaults(func=cmd_restart)

    features = emulator_sub.add_parser("features", help="Show emulator capabilities.")
    features.set_defaults(func=cmd_features)

    emulator.set_defaults(func=cmd_status)


def cmd_reboot(args):
    command = Command.COLDSTART if args.cold else Command.WARMSTART
    async_to_sync(rpc_client(args.socket).call(command))
    return 0


def cmd_status(args):
    st = async_to_sync(rpc_client(args.socket).status())
    machine_name = format_status_name(st.machine_type, STATUS_MACHINE_NAMES)
    sys.stdout.write(
        "paused=%s crashed=%s machine_type=%s emu_ms=%d reset_ms=%d state_seq=%d\n"
        % (
            str(st.paused).lower(),
            str(st.crashed).lower(),
            machine_name,
            st.emu_ms,
            st.reset_ms,
            st.state_seq,
        )
    )
    return 0


def cmd_sysinfo(args):
    info = async_to_sync(rpc_client(args.socket).sysinfo())
    family_name = format_status_name(info.machine_family, STATUS_MACHINE_FAMILY_NAMES)
    os_name = format_status_name(info.os_revision, STATUS_OS_REVISION_NAMES)
    basic_name = format_status_name(info.basic_revision, STATUS_BASIC_REVISION_NAMES)
    builtin_name = format_status_name(
        info.builtin_game_revision, STATUS_BUILTIN_GAME_REVISION_NAMES
    )
    sys.stdout.write(
        "basic_enabled=%s tv_pal=%s machine_family=%s os_revision=%s "
        "basic_revision=%s builtin_game_revision=%s\n"
        % (
            str(info.basic_enabled).lower(),
            str(info.tv_pal).lower(),
            family_name,
            os_name,
            basic_name,
            builtin_name,
        )
    )
    return 0


def cmd_stop(args):
    async_to_sync(rpc_client(args.socket).call(Command.STOP_EMULATOR))
    return 0


def cmd_restart(args):
    async_to_sync(rpc_client(args.socket).call(Command.RESTART_EMULATOR))
    return 0


def cmd_features(args):
    caps = async_to_sync(rpc_client(args.socket).build_features())
    for line in format_capability_lines(caps, EMULATOR_CAPABILITIES):
        sys.stdout.write(line + "\n")
    return 0
