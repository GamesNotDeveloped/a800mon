import curses

from . import debug
from .app import App, StopLoop
from .appstate import state
from .cpustate import CpuStateViewer
from .displaylist import DisplayListViewer
from .rpc import Command, RpcClient
from .screenbuffer import ScreenBufferInspector
from .shortcutbar import ShortcutBar
from .shortcuts import ShortcutLayer, ShortcutManager
from .socket import SocketTransport
from .topbar import TopBar
from .ui import Screen, Window


def main(scr, socket_path):
    wcpu = Window(title="CPU State")
    wdlist = Window(title="DisplayList")
    wscreen = Window(title="Screen Buffer (ATASCII)")
    top = Window(border=False)
    bottom = Window(border=False)

    def init_screen(scr):
        w, h = scr.size
        wcpu.reshape(x=0, y=h - 5, w=w, h=3)
        wdlist.reshape(x=0, y=2, w=40, h=wcpu.y - 3)
        wscreen.reshape(x=wdlist.x + wdlist.w + 2, y=2, w=60, h=wcpu.y - 3)
        top.reshape(x=0, y=0, w=w, h=1)
        bottom.reshape(x=0, y=h - 1, w=w, h=1)

    rpc = RpcClient(SocketTransport(socket_path))

    shortcuts = ShortcutManager()
    def _pause():
        rpc.call(Command.PAUSE)
        shortcuts.touch()

    def _cont():
        rpc.call(Command.CONTINUE)
        shortcuts.touch()

    def _step():
        rpc.call(Command.STEP)
        shortcuts.touch()

    def _step_vblank():
        rpc.call(Command.STEP_VBLANK)
        shortcuts.touch()

    def _coldstart():
        rpc.call(Command.COLDSTART)
        shortcuts.touch()

    def _warmstart():
        rpc.call(Command.WARMSTART)
        shortcuts.touch()

    def _terminate():
        rpc.call(Command.STOP_EMULATOR)
        shortcuts.touch()

    normal = ShortcutLayer("NORMAL")
    normal.add(curses.KEY_F0 + 5, "Step", _step)
    normal.add(curses.KEY_F0 + 6, "Step Vblank", _step_vblank)
    normal.add(curses.KEY_F0 + 8, "Pause", _pause)

    debug = ShortcutLayer("DEBUG")
    debug.add(curses.KEY_F0 + 5, "Step", _step)
    debug.add(curses.KEY_F0 + 6, "Step Vblank", _step_vblank)
    debug.add(curses.KEY_F0 + 8, "Continue", _cont)

    screen_inspector = ScreenBufferInspector(rpc, wscreen)
    display_list = DisplayListViewer(rpc, wdlist)
    cpu = CpuStateViewer(rpc, wcpu)

    import dsm

    transitions = dsm.Transitions(
        (
            ("init", ["normal"], "NORMAL"),
            ("NORMAL", ["pause"], "DEBUG"),
            ("DEBUG", ["cont"], "NORMAL"),
            ("NORMAL", ["menu"], "MENU"),
            ("DEBUG", ["menu"], "MENU"),
            ("MENU", ["esc"], "NORMAL"),
        )
    )
    fsm = dsm.StateMachine(initial="init", transitions=transitions)
    fsm.process("normal")

    menu = ShortcutLayer("SHUTDOWN")
    menu.add("c", "Cold start", _coldstart)
    menu.add("w", "Warm start", _warmstart)
    menu.add("t", "Terminate", _terminate)

    def _enter_menu():
        if fsm.state != "MENU":
            fsm.process("menu")
            shortcuts.set_active_layer(menu)

    def _exit_menu():
        if fsm.state == "MENU":
            fsm.process("esc")
            shortcuts.set_active_layer(normal)

    def _menu_to_normal(action):
        def _wrapped():
            action()
            _exit_menu()
        return _wrapped

    menu.shortcuts[curses.KEY_F0 + 5] = normal.shortcuts[curses.KEY_F0 + 5]
    menu.shortcuts[curses.KEY_F0 + 6] = normal.shortcuts[curses.KEY_F0 + 6]
    menu.add("c", "Cold start", _menu_to_normal(_coldstart))
    menu.add("w", "Warm start", _menu_to_normal(_warmstart))
    menu.add("t", "Terminate", _menu_to_normal(_terminate))

    normal.add(27, "Shutdown", _enter_menu)
    debug.add(27, "Shutdown", _enter_menu)
    menu.add(27, "Back", _exit_menu)

    shortcuts.set_active_layer(normal)
    shortcuts.add_global("q", "Quit", lambda: (_ for _ in ()).throw(StopLoop()))

    def _sync_state(paused):
        if fsm.state == "MENU":
            return
        if paused and fsm.state != "DEBUG":
            fsm.process("pause")
            shortcuts.set_active_layer(debug)
        elif not paused and fsm.state != "NORMAL":
            fsm.process("cont")
            shortcuts.set_active_layer(normal)

    topbar = TopBar(rpc, top, status_hook=_sync_state)
    shortcutbar = ShortcutBar(bottom, shortcuts)

    shortcuts.add_global("d", "DL Inspect", display_list.toggle_inspect)
    scr = Screen(scr, layout_initializer=init_screen)
    app = App(screen=scr)
    app.add_component(shortcuts)
    app.add_component(topbar)
    app.add_component(shortcutbar)
    app.add_component(cpu)
    app.add_component(display_list)
    app.add_component(screen_inspector)

    app.loop()


def run(socket_path="/tmp/atari.sock"):
    try:
        curses.wrapper(lambda scr: main(scr, socket_path))
    except KeyboardInterrupt:
        try:
            curses.endwin()
        except curses.error:
            pass
        raise
    except curses.error:
        try:
            curses.endwin()
        except curses.error:
            pass
        raise
    except Exception:
        try:
            curses.endwin()
        except curses.error:
            pass
        raise
    finally:
        debug.print_log()


if __name__ == "__main__":
    run()
