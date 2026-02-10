import time
import curses

from .app import VisualRpcComponent
from .appstate import state
from .disasm import DecodedInstruction, disasm_6502_decoded
from .rpc import RpcException
from .ui import Color

ASM_COMMENT_COL = 18


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
        self._input_row = 0

    def bind_input(self, screen, input_manager, address_widget, input_row=0):
        self._screen = screen
        self._input_manager = input_manager
        self._address_widget = address_widget
        self._input_row = int(input_row)

    def _start_addr(self) -> int:
        if state.disassembly_addr is None:
            return 0
        return state.disassembly_addr & 0xFFFF

    def update(self):
        if not state.disassembly_enabled:
            return
        if self.window._ih <= 0:
            return
        if state.disassembly_addr is None:
            return
        addr = self._start_addr()
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
        pc = state.cpu.pc & 0xFFFF
        rows = self._lines[: self.window._ih]
        for row_idx, ins in enumerate(rows):
            self.window.cursor = (0, row_idx)
            same_row_as_input = row_idx == self._input_row
            suppress = state.input_focus and same_row_as_input
            rev_attr = curses.A_REVERSE if (ins.addr == pc and not suppress) else 0
            self.window.print(f"{ins.addr:04X}:", attr=Color.ADDRESS.attr() | rev_attr)
            self.window.print(" ", attr=rev_attr)
            self.window.print(f"{ins.raw_text:<8} ", attr=rev_attr)
            self._print_asm(ins, rev_attr)
            self.window.fill_to_eol(attr=rev_attr)
        next_row = len(rows)
        if next_row < self.window._ih:
            self.window.cursor = (0, next_row)
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
            f"{self._start_addr():04X}",
        )
        return True

    def _print_asm(self, ins: DecodedInstruction, rev_attr: int = 0):
        if not ins.mnemonic:
            return
        core_len = len(ins.mnemonic)
        self.window.print(ins.mnemonic, attr=Color.MNEMONIC.attr() | rev_attr)
        if not ins.operand:
            pass
        else:
            self.window.print(" ", attr=rev_attr)
            core_len += 1 + len(ins.operand)
            if ins.flow_target is None or ins.operand_addr_span is None:
                self.window.print(ins.operand, attr=rev_attr)
            else:
                start, end = ins.operand_addr_span
                self.window.print(ins.operand[:start], attr=rev_attr)
                self.window.print(
                    ins.operand[start:end], attr=Color.ADDRESS.attr() | rev_attr
                )
                self.window.print(ins.operand[end:], attr=rev_attr)
        if not ins.comment:
            return
        if core_len < ASM_COMMENT_COL:
            self.window.print(" " * (ASM_COMMENT_COL - core_len), attr=rev_attr)
        self.window.print(" ", attr=rev_attr)
        self.window.print(ins.comment, attr=Color.COMMENT.attr() | rev_attr)
