import curses

from . import debug
from .app import App, Component, StopLoop
from .appstate import AppMode, shortcuts, state
from .cpustate import CpuStateViewer
from .displaylist import DisplayListViewer
from .rpc import Command, RpcClient, RpcException
from .screenbuffer import ScreenBufferInspector
from .shortcutbar import ShortcutBar
from .shortcuts import Shortcut, ShortcutLayer
from .socket import SocketTransport
from .topbar import TopBar
from .ui import Screen, Window


class AppModeUpdater(Component):
    def update(self):
        if state.active_mode in (AppMode.DEBUG, AppMode.NORMAL):
            state.active_mode = AppMode.DEBUG if state.paused else AppMode.NORMAL


def exit_app():
    raise StopLoop


def build_shortcuts(rpc):
    def call_rpc(command):
        def fn():
            try:
                rpc.call(command)
            except RpcException:
                pass

        return fn

    def exit_shutdown_mode():
        state.active_mode = AppMode.DEBUG if state.paused else AppMode.NORMAL

    def enter_shutdown_mode():
        state.active_mode = AppMode.SHUTDOWN

    def change_mode(mode: AppMode, callback):
        def fn():
            state.active_mode = mode
            callback()

        return fn

    step = Shortcut(curses.KEY_F0 + 5, "Step", call_rpc(Command.STEP))
    step_vblank = Shortcut(
        curses.KEY_F0 + 6, "Step VBLANK", call_rpc(Command.STEP_VBLANK)
    )
    pause = Shortcut(
        curses.KEY_F0 +
        8, "Pause", change_mode(AppMode.DEBUG, call_rpc(Command.PAUSE))
    )
    cont = Shortcut(
        curses.KEY_F0 + 8,
        "Continue",
        change_mode(AppMode.NORMAL, call_rpc(Command.CONTINUE)),
    )
    enter_shutdown = Shortcut(27, "Shutdown", enter_shutdown_mode)
    exit_shutdown = Shortcut(27, "Back", exit_shutdown_mode)

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

    def back_to_main(callback):
        def fn():
            callback()
            exit_shutdown_mode()

        return fn

    coldstart = Shortcut("c", "Cold start", back_to_main(
        call_rpc(Command.COLDSTART)))
    warmstart = Shortcut("w", "Warm start", back_to_main(
        call_rpc(Command.WARMSTART)))
    terminate = Shortcut(
        "t", "Terminate", back_to_main(call_rpc(Command.STOP_EMULATOR))
    )

    shutdown = ShortcutLayer("SHUTDOWN")
    shutdown.add(coldstart)
    shutdown.add(warmstart)
    shutdown.add(terminate)
    shutdown.add(exit_shutdown)

    shortcuts.add(AppMode.NORMAL, normal)
    shortcuts.add(AppMode.DEBUG, debug)
    shortcuts.add(AppMode.SHUTDOWN, shutdown)

    shortcuts.add_global(Shortcut("q", "Quit", exit_app))

    return shortcuts


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

    screen_inspector = ScreenBufferInspector(rpc, wscreen)
    display_list = DisplayListViewer(rpc, wdlist)
    cpu = CpuStateViewer(rpc, wcpu)

    appmode_updater = AppModeUpdater()
    topbar = TopBar(rpc, top)
    shortcutbar = ShortcutBar(bottom)
    scr = Screen(scr, layout_initializer=init_screen)
    app = App(screen=scr)

    app.add_component(appmode_updater)
    app.add_component(topbar)
    app.add_component(shortcutbar)
    app.add_component(cpu)
    app.add_component(display_list)
    app.add_component(screen_inspector)

    build_shortcuts(rpc)
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
