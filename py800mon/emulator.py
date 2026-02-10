CAP_MONITOR_BREAKPOINTS = 0x0008

EMULATOR_CAPABILITIES = [
    (0x0001, "SDL2 video backend (VIDEO_SDL2)"),
    (0x0002, "SDL1 video backend (VIDEO_SDL)"),
    (0x0003, "Sound support (SOUND)"),
    (0x0004, "Callback sound backend (SOUND_CALLBACK)"),
    (0x0005, "Audio recording (AUDIO_RECORDING)"),
    (0x0006, "Video recording (VIDEO_RECORDING)"),
    (0x0007, "Code breakpoints/history (MONITOR_BREAK)"),
    (0x0008, "User breakpoint table (MONITOR_BREAKPOINTS)"),
    (0x0009, "Readline monitor support (MONITOR_READLINE)"),
    (0x000A, "Disassembler label hints (MONITOR_HINTS)"),
    (0x000B, "UTF-8 monitor output (MONITOR_UTF8)"),
    (0x000C, "ANSI monitor output (MONITOR_ANSI)"),
    (0x000D, "Monitor assembler command (MONITOR_ASSEMBLER)"),
    (0x000E, "Monitor profiling/coverage (MONITOR_PROFILE)"),
    (0x000F, "Monitor TRACE command (MONITOR_TRACE)"),
    (0x0010, "NetSIO/FujiNet emulation (NETSIO)"),
    (0x0011, "IDE emulation (IDE)"),
    (0x0012, "R: device support (R_IO_DEVICE)"),
    (0x0013, "Black Box emulation (PBI_BB)"),
    (0x0014, "MIO emulation (PBI_MIO)"),
    (0x0015, "Prototype80 emulation (PBI_PROTO80)"),
    (0x0016, "1400XL/1450XLD emulation (PBI_XLD)"),
    (0x0017, "VoiceBox emulation (VOICEBOX)"),
    (0x0018, "AF80 card emulation (AF80)"),
    (0x0019, "BIT3 card emulation (BIT3)"),
    (0x001A, "XEP80 emulation (XEP80_EMULATION)"),
    (0x001B, "NTSC filter (NTSC_FILTER)"),
    (0x001C, "PAL blending (PAL_BLENDING)"),
    (0x001D, "Crash menu support (CRASH_MENU)"),
    (0x001E, "New cycle-exact core (NEW_CYCLE_EXACT)"),
    (0x001F, "libpng support (HAVE_LIBPNG)"),
    (0x0020, "zlib support (HAVE_LIBZ)"),
]


def format_on_off_badge(enabled: bool, use_color: bool = False) -> str:
    text = "ON " if enabled else "OFF"
    badge = f" {text} "
    if not use_color:
        return badge
    if enabled:
        return f"\x1b[42;30m{badge}\x1b[0m"
    return f"\x1b[41;97;1m{badge}\x1b[0m"


def format_config_lines(cap_ids, use_color: bool = False):
    enabled = set(int(v) & 0xFFFF for v in cap_ids)
    known = set()
    lines = []
    for cap_id, desc in EMULATOR_CAPABILITIES:
        known.add(cap_id)
        badge = format_on_off_badge(cap_id in enabled, use_color=use_color)
        lines.append(f"{badge} {desc}")
    for cap_id in sorted(v for v in enabled if v not in known):
        badge = format_on_off_badge(True, use_color=use_color)
        lines.append(f"{badge} Unknown capability 0x{cap_id:04X}")
    return lines
