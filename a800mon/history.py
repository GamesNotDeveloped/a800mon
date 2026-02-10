import curses
import time

from .app import VisualRpcComponent
from .appstate import state
from .disasm import disasm_6502_one_parts
from .rpc import RpcException
from .ui import Color

FLOW_MNEMONICS = {
    "JMP",
    "JSR",
    "BCC",
    "BCS",
    "BEQ",
    "BMI",
    "BNE",
    "BPL",
    "BVC",
    "BVS",
    "BRA",
}


class HistoryViewer(VisualRpcComponent):
    def __init__(self, *args, **kwargs):
        self._update_interval = kwargs.pop("update_interval", 0.05)
        super().__init__(*args, **kwargs)
        self._last_update = 0.0
        self._entries = []
        self._can_disasm = True
        self._last_title_pc = None

    def update(self):
        now = time.time()
        if self._last_update and now - self._last_update < self._update_interval:
            return
        try:
            self._entries = self.rpc.history()
            self._last_update = now
        except RpcException:
            pass

    def render(self, force_redraw=False):
        title_pc = state.cpu.pc
        if title_pc != self._last_title_pc:
            self._last_title_pc = title_pc
            self.window.set_title(f"History [PC: {title_pc:04X}]")

        self.window.cursor = 0, 0
        if self.window._ih <= 0:
            return

        rows = self._entries[-self.window._ih:]
        for idx, entry in enumerate(rows):
            rev_attr = curses.A_REVERSE if idx == 0 else 0
            raw_text, asm_text = self._format_disasm(entry.pc, entry.opbytes)
            self.window.print(f"{entry.pc:04X}:", attr=Color.ADDRESS.attr() | rev_attr)
            self.window.print(" ", attr=rev_attr)
            self.window.print(f"{raw_text:<8} ", attr=rev_attr)
            self._print_asm(asm_text, rev_attr)
            self.window.clear_to_eol(inverse=(idx == 0))
            self.window.newline()
        self.window.clear_to_bottom()

    def _format_disasm(self, pc: int, opbytes: bytes) -> tuple[str, str]:
        if self._can_disasm:
            try:
                return disasm_6502_one_parts(pc, opbytes)
            except RuntimeError:
                self._can_disasm = False
        return " ".join(f"{b:02X}" for b in opbytes), ""

    def _print_asm(self, asm_text: str, rev_attr: int = 0):
        if not asm_text:
            return
        parts = asm_text.split(None, 1)
        mnemonic = parts[0].upper()
        operand = parts[1] if len(parts) > 1 else ""
        self.window.print(mnemonic, attr=Color.MNEMONIC.attr() | rev_attr)
        if not operand:
            return
        self.window.print(" ", attr=rev_attr)
        if mnemonic not in FLOW_MNEMONICS:
            self.window.print(operand, attr=rev_attr)
            return
        span = _find_hex_addr_span(operand)
        if span is None:
            self.window.print(operand, attr=rev_attr)
            return
        start, end = span
        self.window.print(operand[:start], attr=rev_attr)
        self.window.print(operand[start:end], attr=Color.ADDRESS.attr() | rev_attr)
        self.window.print(operand[end:], attr=rev_attr)


def _find_hex_addr_span(text: str):
    start = text.find("$")
    if start < 0:
        return None
    end = start + 1
    while end < len(text):
        ch = text[end]
        if ("0" <= ch <= "9") or ("A" <= ch <= "F") or ("a" <= ch <= "f"):
            end += 1
            continue
        break
    if end == start + 1:
        return None
    return start, end
