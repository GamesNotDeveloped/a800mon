import dataclasses

from .datastructures import CpuState, DisplayList, ScreenBuffer


@dataclasses.dataclass
class AppState:
    dlist: DisplayList
    screen_buffer: ScreenBuffer
    cpu: CpuState
    monitor_frame_time_ms: int


state = AppState(
    dlist=DisplayList(),
    screen_buffer=ScreenBuffer(),
    cpu=CpuState(),
    monitor_frame_time_ms=0,
)
