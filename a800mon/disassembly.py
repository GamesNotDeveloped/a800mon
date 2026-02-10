import time

from .app import VisualRpcComponent
from .appstate import state
from .disasm import disasm_6502
from .rpc import RpcException
from .ui import Color


class DisassemblyViewer(VisualRpcComponent):
    def __init__(self, *args, **kwargs):
        self._update_interval = kwargs.pop("update_interval", 0.05)
        super().__init__(*args, **kwargs)
        self._last_update = 0.0
        self._last_addr = None
        self._lines = []

    def update(self):
        if not state.disassembly_enabled:
            return
        if self.window._ih <= 0:
            return
        addr = state.disassembly_addr & 0xFFFF
        now = time.time()
        if (
            self._last_addr == addr
            and self._last_update
            and now - self._last_update < self._update_interval
        ):
            return
        read_len = max(3, self.window._ih * 3)
        try:
            data = self.rpc.read_memory(addr, read_len)
        except RpcException:
            return
        self._lines = disasm_6502(addr, data)[: self.window._ih]
        self._last_addr = addr
        self._last_update = now

    def render(self, force_redraw=False):
        self.window.cursor = 0, 0
        for line in self._lines[: self.window._ih]:
            if ":" in line:
                addr, rest = line.split(":", 1)
                self.window.print(f"{addr}:", attr=Color.ADDRESS.attr())
                self.window.print(rest)
            else:
                self.window.print(line)
            self.window.clear_to_eol()
            self.window.newline()
        self.window.clear_to_bottom()
