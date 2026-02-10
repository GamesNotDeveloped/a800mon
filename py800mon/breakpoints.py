from .app import VisualRpcComponent
from .appstate import state
from .datastructures import BreakpointClauseEntry, BreakpointConditionEntry
from .rpc import RpcException
from .ui import Color

BP_TYPE_NAMES = {
    1: "pc",
    2: "a",
    3: "x",
    4: "y",
    5: "s",
    6: "read",
    7: "write",
    8: "access",
}

BP_OP_NAMES = {
    1: "<",
    2: "<=",
    3: "==",
    4: "!=",
    5: ">=",
    6: ">",
}


class BreakpointsViewer(VisualRpcComponent):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._last_snapshot = None
        self._last_state_seq = None
        self._screen = None
        self._dispatcher = None

    def bind_input(self, screen):
        self._screen = screen

    async def update(self):
        if not state.breakpoints_supported:
            return False
        cur_seq = state.state_seq
        if self._last_state_seq == cur_seq and self._last_snapshot is not None:
            return False
        self._last_state_seq = cur_seq
        try:
            enabled, clauses = await self.rpc.breakpoint_list()
        except RpcException:
            return False
        parsed = []
        for clause in clauses:
            conds = []
            for cond_type, op, addr, value in clause:
                conds.append(
                    BreakpointConditionEntry(
                        cond_type=cond_type & 0xFF,
                        op=op & 0xFF,
                        addr=addr & 0xFFFF,
                        value=value & 0xFFFF,
                    )
                )
            parsed.append(BreakpointClauseEntry(conditions=tuple(conds)))
        snapshot = (bool(enabled), tuple(parsed))
        if snapshot == self._last_snapshot:
            return False
        self._last_snapshot = snapshot
        if (
            self._dispatcher is not None
            and (state.breakpoints_enabled != enabled or state.breakpoints != parsed)
        ):
            self._dispatcher.update_breakpoints(enabled, parsed)
        return True

    def attach_dispatcher(self, dispatcher):
        self._dispatcher = dispatcher

    def render(self, force_redraw=False):
        ih = self.window._ih
        if ih <= 0:
            return
        self.window.cursor = (0, 0)
        self.window.set_tag_active("bp_enabled", state.breakpoints_enabled)

        max_rows = ih
        if not state.breakpoints:
            if max_rows > 0:
                self.window.print("No breakpoint clauses.", attr=Color.COMMENT.attr())
                self.window.clear_to_eol()
                self.window.newline()
                self.window.clear_to_bottom()
            return

        for idx, clause in enumerate(state.breakpoints[:max_rows], start=1):
            self.window.print(f"#{idx:02d} ", attr=Color.ADDRESS.attr())
            self._print_clause(clause)
            self.window.clear_to_eol()
            self.window.newline()
        self.window.clear_to_bottom()

    def _print_clause(self, clause: BreakpointClauseEntry):
        for idx, cond in enumerate(clause.conditions):
            if idx:
                self.window.print(" && ", attr=Color.TEXT.attr())
            self._print_condition(cond)

    def _print_condition(self, cond: BreakpointConditionEntry):
        op = BP_OP_NAMES.get(cond.op, f"op{cond.op}")
        if cond.cond_type == 9:
            self.window.print("mem[", attr=Color.TEXT.attr())
            self.window.print(f"{cond.addr:04X}", attr=Color.ADDRESS.attr())
            self.window.print("]", attr=Color.TEXT.attr())
        else:
            name = BP_TYPE_NAMES.get(cond.cond_type, f"type{cond.cond_type}")
            self.window.print(name, attr=Color.TEXT.attr())
        self.window.print(f" {op} ", attr=Color.TEXT.attr())
        if cond.cond_type in (2, 3, 4, 5):
            self.window.print(f"{cond.value:02X}", attr=Color.ADDRESS.attr())
        else:
            self.window.print(f"{cond.value:04X}", attr=Color.ADDRESS.attr())

    def handle_input(self, ch):
        if self._screen is None:
            return False
        if self._screen.focused is not self.window:
            return False
        return False
