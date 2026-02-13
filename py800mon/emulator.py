CAP_MONITOR_BREAK = 0x0001
CAP_MONITOR_BREAKPOINTS = 0x0002
CAP_EMULATION_ATARI_800 = 0x0003
CAP_EMULATION_XE_XL = 0x0004
CAP_EMULATION_ATARI_5200 = 0x0005

STATUS_MACHINE_ATARI_800 = 0
STATUS_MACHINE_XL_XE = 1
STATUS_MACHINE_ATARI_5200 = 2
STATUS_MACHINE_ATARI_1200XL = 3
STATUS_MACHINE_ATARI_800XL = 4
STATUS_MACHINE_ATARI_130XE = 5
STATUS_MACHINE_ATARI_320XE_COMPY_SHOP = 6
STATUS_MACHINE_ATARI_320XE_RAMBO = 7
STATUS_MACHINE_ATARI_576XE = 8
STATUS_MACHINE_ATARI_1088XE = 9
STATUS_MACHINE_ATARI_XEGS = 10
STATUS_MACHINE_ATARI_400 = 11
STATUS_MACHINE_ATARI_600XL = 12

STATUS_MACHINE_NAMES = {
    STATUS_MACHINE_ATARI_800: "atari800",
    STATUS_MACHINE_XL_XE: "xl_xe",
    STATUS_MACHINE_ATARI_5200: "atari5200",
    STATUS_MACHINE_ATARI_1200XL: "atari1200xl",
    STATUS_MACHINE_ATARI_800XL: "atari800xl",
    STATUS_MACHINE_ATARI_130XE: "atari130xe",
    STATUS_MACHINE_ATARI_320XE_COMPY_SHOP: "atari320xe_compy_shop",
    STATUS_MACHINE_ATARI_320XE_RAMBO: "atari320xe_rambo",
    STATUS_MACHINE_ATARI_576XE: "atari576xe",
    STATUS_MACHINE_ATARI_1088XE: "atari1088xe",
    STATUS_MACHINE_ATARI_XEGS: "atarixegs",
    STATUS_MACHINE_ATARI_400: "atari400",
    STATUS_MACHINE_ATARI_600XL: "atari600xl",
}

STATUS_MACHINE_FAMILY_NAMES = {
    0: "atari800",
    1: "xl_xe",
    2: "atari5200",
}

STATUS_OS_REVISION_NAMES = {
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

STATUS_BASIC_REVISION_NAMES = {
    0x00: "a",
    0x01: "b",
    0x02: "c",
    0x03: "custom",
    0x04: "altirra",
    0xFF: "none",
}

STATUS_BUILTIN_GAME_REVISION_NAMES = {
    0x00: "orig",
    0x01: "custom",
    0xFF: "none",
}


def format_status_name(value: int, mapping: dict[int, str]) -> str:
    name = mapping.get(value)
    if name is None:
        return "unknown"
    return name

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
