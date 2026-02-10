import time

from .app import VisualRpcComponent
from .appstate import state
from .disasm import DecodedInstruction, disasm_6502_decoded
from .rpc import RpcException
from .ui import Color


class DisassemblyViewer(VisualRpcComponent):
    def __init__(self, *args, **kwargs):
        self._update_interval = kwargs.pop("update_interval", 0.05)
        super().__init__(*args, **kwargs)
        self._last_update = 0.0
        self._last_addr = None
        self._lines = []
        self._screen = None
        self._input_manager = None
        self._address_widget = None

    def bind_input(self, screen, input_manager, address_widget):
        self._screen = screen
        self._input_manager = input_manager
        self._address_widget = address_widget

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
        self._lines = disasm_6502_decoded(addr, data)[: self.window._ih]
        self._last_addr = addr
        self._last_update = now

    def render(self, force_redraw=False):
        self.window.cursor = 0, 0
        for ins in self._lines[: self.window._ih]:
            self.window.print(f"{ins.addr:04X}:", attr=Color.ADDRESS.attr())
            self.window.print(" ")
            self.window.print(f"{ins.raw_text:<8} ")
            self._print_asm(ins)
            self.window.clear_to_eol()
            self.window.newline()
        self.window.clear_to_bottom()

    def handle_input(self, ch):
        if state.input_focus:
            return False
        if ch != ord("/"):
            return False
        if not self.window.visible:
            return False
        if self._screen is None or self._screen.focused is not self.window:
            return False
        if self._input_manager is None or self._address_widget is None:
            return False
        self._input_manager.open(
            self._address_widget,
            f"{state.disassembly_addr & 0xFFFF:04X}",
        )
        return True

    def _print_asm(self, ins: DecodedInstruction):
        if not ins.mnemonic:
            return
        self.window.print(ins.mnemonic, attr=Color.MNEMONIC.attr())
        if not ins.operand:
            return
        self.window.print(" ")
        if ins.flow_target is None or ins.operand_addr_span is None:
            self.window.print(ins.operand)
        else:
            start, end = ins.operand_addr_span
            self.window.print(ins.operand[:start])
            self.window.print(ins.operand[start:end], attr=Color.ADDRESS.attr())
            self.window.print(ins.operand[end:])
        if not ins.comment:
            return
        self.window.print(" ")
        self.window.print(ins.comment)
