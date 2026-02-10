import curses

from .app import VisualRpcComponent
from .appstate import state
from .disasm import DecodedInstruction, disasm_6502_one_decoded
from .rpc import RpcException
from .ui import Color

ASM_COMMENT_COL = 18


class HistoryViewer(VisualRpcComponent):
    def __init__(self, *args, **kwargs):
        self._reverse_order = bool(kwargs.pop("reverse_order", False))
        super().__init__(*args, **kwargs)
        self._entries = []
        self._can_disasm = True
        self._next_pc = None
        self._next_opbytes = b""
        self._last_snapshot = None
        self._render_cache = []
        self._rendered_rows = 0
        self._decoded_cache = {}

    async def update(self):
        try:
            entries = await self.rpc.history()
            pc = state.cpu.pc & 0xFFFF
            next_opbytes = await self.rpc.read_memory(pc, 3)
            snapshot = (tuple(entries), pc, next_opbytes)
            if self._last_snapshot == snapshot:
                return False
            self._last_snapshot = snapshot
            self._entries = entries
            self._next_pc = pc
            self._next_opbytes = next_opbytes
            return True
        except RpcException:
            return False

    def render(self, force_redraw=False):
        ih = self.window._ih
        if ih <= 0:
            return
        if len(self._render_cache) != ih:
            self._render_cache = [None] * ih
            force_redraw = True

        next_pc = state.cpu.pc & 0xFFFF
        next_bytes = self._next_opbytes if self._next_pc == next_pc else b""
        next_ins = self._format_disasm_cached(next_pc, next_bytes)

        render_rows = []
        if self._reverse_order:
            rows = list(reversed(self._entries[: max(0, ih - 1)]))
            for entry in rows:
                render_rows.append(
                    (entry.pc, self._format_disasm_cached(entry.pc, entry.opbytes), 0)
                )
            render_rows.append((next_pc, next_ins, curses.A_REVERSE))
        else:
            render_rows.append((next_pc, next_ins, curses.A_REVERSE))
            if ih > 1:
                rows = self._entries[: ih - 1]
                for entry in rows:
                    render_rows.append(
                        (entry.pc, self._format_disasm_cached(entry.pc, entry.opbytes), 0)
                    )

        for row_idx, (pc, ins, rev_attr) in enumerate(render_rows):
            row_sig = (pc, ins, rev_attr)
            if not force_redraw and self._render_cache[row_idx] == row_sig:
                continue
            self.window.cursor = (0, row_idx)
            self._print_row(pc, ins, rev_attr)
            if rev_attr:
                self.window.fill_to_eol(attr=rev_attr)
            else:
                self.window.clear_to_eol()
            self._render_cache[row_idx] = row_sig

        next_row = len(render_rows)
        if next_row < ih and (force_redraw or self._rendered_rows != next_row):
            self.window.cursor = (0, next_row)
            self.window.clear_to_bottom()
        for idx in range(next_row, ih):
            self._render_cache[idx] = None
        self._rendered_rows = next_row

    def _format_disasm(self, pc: int, opbytes: bytes) -> DecodedInstruction:
        if self._can_disasm:
            try:
                ins = disasm_6502_one_decoded(pc, opbytes)
                if ins is not None:
                    return ins
            except RuntimeError:
                self._can_disasm = False
        raw_text = " ".join(f"{b:02X}" for b in opbytes)
        return DecodedInstruction(
            addr=pc & 0xFFFF,
            size=len(opbytes),
            raw=opbytes,
            raw_text=raw_text,
            mnemonic="",
            operand="",
            comment="",
            asm_text="",
            addressing="",
            flow_target=None,
            operand_addr_span=None,
        )

    def _format_disasm_cached(self, pc: int, opbytes: bytes) -> DecodedInstruction:
        key = (pc & 0xFFFF, bytes(opbytes))
        if key in self._decoded_cache:
            return self._decoded_cache[key]
        ins = self._format_disasm(pc, opbytes)
        self._decoded_cache[key] = ins
        if len(self._decoded_cache) > 2048:
            self._decoded_cache.clear()
        return ins

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
            self.window.print(
                " " * (ASM_COMMENT_COL - core_len), attr=rev_attr)
        self.window.print(" ", attr=rev_attr)
        self.window.print(ins.comment, attr=Color.COMMENT.attr() | rev_attr)

    def _print_row(
        self, pc: int, ins: DecodedInstruction, rev_attr: int, prefix: str = ""
    ):
        if prefix:
            self.window.print(prefix, attr=rev_attr)
        self.window.print(f"{pc:04X}:", attr=Color.ADDRESS.attr() | rev_attr)
        self.window.print(" ", attr=rev_attr)
        self.window.print(f"{ins.raw_text:<8} ", attr=rev_attr)
        self._print_asm(ins, rev_attr)
