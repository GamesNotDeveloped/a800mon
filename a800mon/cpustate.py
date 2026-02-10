from .app import VisualRpcComponent
from .appstate import state, store
from .datastructures import CpuState
from .rpc import Command, RpcException


class CpuStateViewer(VisualRpcComponent):
    def update(self):
        try:
            data = self.rpc.cpu_state()
        except RpcException:
            pass
        else:
            ypos, xpos, pc, a, x, y, s, p = data
            cpu = CpuState(ypos=ypos, xpos=xpos, pc=pc, a=a, x=x, y=y, s=s, p=p)
            store.set_cpu(cpu)

    def render(self, force_redraw=False):
        self.window.print_line(repr(state.cpu))

    def handle_input(self, ch):
        if ch == ord("p"):
            self.rpc.call(Command.PAUSE)
        if ch == ord("s"):
            self.rpc.call(Command.STEP)
        if ch == ord("v"):
            self.rpc.call(Command.STEP_VBLANK)
        if ch == ord("c"):
            self.rpc.call(Command.CONTINUE)
