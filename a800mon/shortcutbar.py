from .app import VisualComponent
import curses

from .shortcuts import Shortcut
from .ui import Color


class ShortcutBar(VisualComponent):
    LAYER_WIDTH = 16
    SLOT_WIDTH = 16
    SLOT_GAP = 1

    def __init__(self, window, manager):
        super().__init__(window)
        self._manager = manager
        self._last_version = None

    def _key_text(self, key):
        if key == 27:
            return "Esc"
        if key >= curses.KEY_F0 and key <= curses.KEY_F0 + 63:
            return f"F{key - curses.KEY_F0}"
        if 0 <= key < 32:
            return "^" + chr(key + 64)
        if 32 <= key <= 126:
            return chr(key).upper()
        return str(key)

    def _print_slot(self, shortcut: Shortcut, attr):
        key_text = f" {self._key_text(shortcut.key)} "
        label_text = shortcut.label
        text = f"{key_text} {label_text}"
        if len(text) > self.SLOT_WIDTH:
            text = text[: self.SLOT_WIDTH]
        elif len(text) < self.SLOT_WIDTH:
            text = text.ljust(self.SLOT_WIDTH)

        inv_attr = attr | curses.A_REVERSE
        key_len = len(key_text)
        for idx, ch in enumerate(text):
            if idx < key_len:
                self.window.print_char(ch, attr=inv_attr)
            else:
                self.window.print_char(ch, attr=attr)

    def render(self, force_redraw=False):
        shortcuts = self._manager.hints(show_globals=True)
        if (
            not force_redraw
            and self._last_version == self._manager.version
            and getattr(self, "_last_labels", None) == shortcuts
        ):
            return
        self._last_version = self._manager.version
        self._last_labels = shortcuts

        self.window.cursor = 0, 0
        self.window.clear_to_eol()

        attr = Color.TEXT.attr()
        layer = self._manager.current_layer()
        layer_label = layer.name if layer else ""
        layer_text = layer_label[: self.LAYER_WIDTH].ljust(self.LAYER_WIDTH)
        inv_attr = attr | curses.A_REVERSE
        for ch in layer_text:
            self.window.print_char(ch, attr=inv_attr)

        globals_only = self._manager.visible_globals()
        locals_only = self._manager.visible_locals()

        left_slots = []
        for sc in locals_only:
            left_slots.append(sc)
        if left_slots:
            self.window.print(" " * self.SLOT_GAP, attr=attr)
            for idx, shortcut in enumerate(left_slots):
                if idx:
                    self.window.print(" " * self.SLOT_GAP, attr=attr)
                self._print_slot(shortcut, attr)

        # Right-align globals.
        if globals_only:
            total_globals = (
                len(globals_only) * self.SLOT_WIDTH
                + self.SLOT_GAP * max(0, len(globals_only) - 1)
            )
            right_start = max(self.window._iw - total_globals, 0)
            # If right area overlaps left, let left win; globals will be clipped.
            self.window.cursor = right_start, 0
            for idx, shortcut in enumerate(globals_only):
                if idx:
                    self.window.print(" " * self.SLOT_GAP, attr=attr)
                self._print_slot(shortcut, attr)
        self.window.fill_to_eol(attr=attr)
