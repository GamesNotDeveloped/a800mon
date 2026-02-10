import curses

from . import debug
from .actions import ActionDispatcher, Actions, ShortcutInput
from .app import App, Component
from .appstate import AppMode, shortcuts, state
from .cpustate import CpuStateViewer
from .displaylist import DisplayListViewer
from .rpc import RpcClient
from .screenbuffer import ScreenBufferInspector
from .shortcutbar import ShortcutBar
from .shortcuts import Shortcut, ShortcutLayer
from .socket import SocketTransport
from .topbar import TopBar
from .ui import Screen, Window


class AppModeUpdater(Component):
    def __init__(self, dispatcher):
        self._dispatcher = dispatcher
        self._last_paused = None

    def update(self):
        if self._last_paused is None:
            self._last_paused = state.paused
            self._dispatcher.dispatch(Actions.SYNC_MODE)
            return
        if state.paused != self._last_paused:
            self._last_paused = state.paused
            self._dispatcher.dispatch(Actions.SYNC_MODE)


def main(scr, socket_path):
    rpc = RpcClient(SocketTransport(socket_path))
    dispatcher = ActionDispatcher(rpc)

    wcpu = Window(title="CPU State")
    wdlist = Window(title="DisplayList")
    wscreen = Window(title="Screen Buffer (ATASCII)")
    top = Window(border=False)
    bottom = Window(border=False)

    screen_inspector = ScreenBufferInspector(rpc, wscreen)
    display_list = DisplayListViewer(rpc, wdlist)
    cpu = CpuStateViewer(rpc, wcpu)
    topbar = TopBar(rpc, top)
    appmode_updater = AppModeUpdater(dispatcher)
    shortcutbar = ShortcutBar(bottom)

    def init_screen(scr):
        w, h = scr.size
        wcpu.reshape(x=0, y=h - 5, w=w, h=3)
        wdlist.reshape(x=0, y=2, w=40, h=wcpu.y - 3)
        wscreen.reshape(x=wdlist.x + wdlist.w + 2, y=2, w=60, h=wcpu.y - 3)
        top.reshape(x=0, y=0, w=w, h=1)
        bottom.reshape(x=0, y=h - 1, w=w, h=1)

    def build_shortcuts():
        step = Shortcut(
            curses.KEY_F0 +
            5, "Step", lambda: dispatcher.dispatch(Actions.STEP)
        )
        step_vblank = Shortcut(
            curses.KEY_F0 + 6,
            "Step VBLANK",
            lambda: dispatcher.dispatch(Actions.STEP_VBLANK),
        )
        pause = Shortcut(
            curses.KEY_F0 + 8,
            "Pause",
            lambda: dispatcher.dispatch(Actions.PAUSE),
        )
        cont = Shortcut(
            curses.KEY_F0 + 8,
            "Continue",
            lambda: dispatcher.dispatch(Actions.CONTINUE),
        )
        enter_shutdown = Shortcut(
            27, "Shutdown", lambda: dispatcher.dispatch(Actions.ENTER_SHUTDOWN)
        )
        exit_shutdown = Shortcut(
            27, "Back", lambda: dispatcher.dispatch(Actions.EXIT_SHUTDOWN)
        )

        normal = ShortcutLayer("NORMAL")
        normal.add(step)
        normal.add(step_vblank)
        normal.add(pause)
        normal.add(enter_shutdown)

        debug = ShortcutLayer("DEBUG")
        debug.add(step)
        debug.add(step_vblank)
        debug.add(cont)
        debug.add(enter_shutdown)

        coldstart = Shortcut(
            "c", "Cold start", lambda: dispatcher.dispatch(Actions.COLDSTART)
        )
        warmstart = Shortcut(
            "w", "Warm start", lambda: dispatcher.dispatch(Actions.WARMSTART)
        )
        terminate = Shortcut(
            "t", "Terminate", lambda: dispatcher.dispatch(Actions.TERMINATE)
        )

        shutdown = ShortcutLayer("SHUTDOWN")
        shutdown.add(coldstart)
        shutdown.add(warmstart)
        shutdown.add(terminate)
        shutdown.add(exit_shutdown)

        shortcuts.add(AppMode.NORMAL, normal)
        shortcuts.add(AppMode.DEBUG, debug)
        shortcuts.add(AppMode.SHUTDOWN, shutdown)

        shortcuts.add_global(
            Shortcut(
                "d",
                "Toggle DLIST",
                lambda: dispatcher.dispatch(Actions.TOGGLE_DLIST_INSPECT),
            )
        )
        shortcuts.add_global(
            Shortcut("q", "Quit", lambda: dispatcher.dispatch(Actions.QUIT))
        )

    app = App(screen=Screen(scr, layout_initializer=init_screen))

    input_processor = ShortcutInput(shortcuts, dispatcher)
    app.add_component(dispatcher)
    app.add_component(input_processor)
    app.add_component(topbar)
    app.add_component(appmode_updater)
    app.add_component(shortcutbar)
    app.add_component(cpu)
    app.add_component(display_list)
    app.add_component(screen_inspector)

    build_shortcuts()
    app.loop()


def run(socket_path):
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
