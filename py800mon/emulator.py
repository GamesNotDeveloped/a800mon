CAP_MONITOR_BREAK = 0x0001
CAP_MONITOR_BREAKPOINTS = 0x0002
CAP_EMULATION_ATARI_800 = 0x0003
CAP_EMULATION_XE_XL = 0x0004
CAP_EMULATION_ATARI_5200 = 0x0005

STATUS_MACHINE_ATARI_800 = 0
STATUS_MACHINE_XL_XE = 1
STATUS_MACHINE_ATARI_5200 = 2

STATUS_MACHINE_NAMES = {
    STATUS_MACHINE_ATARI_800: "atari800",
    STATUS_MACHINE_XL_XE: "xl_xe",
    STATUS_MACHINE_ATARI_5200: "atari5200",
}

EMULATOR_CAPABILITIES = [
    (CAP_MONITOR_BREAK, "Code breakpoints/history enabled (MONITOR_BREAK)."),
    (
        CAP_MONITOR_BREAKPOINTS,
        "User breakpoint table enabled (MONITOR_BREAKPOINTS).",
    ),
    (
        CAP_EMULATION_ATARI_800,
        "Atari 400/800 emulation available in this build.",
    ),
    (CAP_EMULATION_XE_XL, "Atari XL/XE emulation available in this build."),
    (CAP_EMULATION_ATARI_5200, "Atari 5200 emulation available in this build."),
]
