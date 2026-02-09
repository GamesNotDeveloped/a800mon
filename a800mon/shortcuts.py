import dataclasses


def _normalize_key(key):
    if isinstance(key, str):
        if len(key) != 1:
            raise ValueError("Shortcut key string must be a single character.")
        return ord(key)
    return int(key)


@dataclasses.dataclass
class Shortcut:
    key: int
    label: str
    callback: callable


class ShortcutLayer:
    def __init__(self, name=""):
        self.name = name
        self.shortcuts = {}

    def add(self, key, label, callback):
        k = _normalize_key(key)
        self.shortcuts[k] = Shortcut(
            key=k,
            label=str(label),
            callback=callback,
        )
        return self


class ShortcutManager:
    def __init__(self):
        self._globals = {}
        self._layers = []
        self._version = 0

    def update(self):
        pass

    def handle_input(self, ch):
        if ch in self._globals:
            self._globals[ch].callback()
            return True
        layer = self.current_layer()
        if layer and ch in layer.shortcuts:
            layer.shortcuts[ch].callback()
            return True
        return False

    @property
    def version(self):
        return self._version

    def _bump(self):
        self._version += 1
        return self._version

    def touch(self):
        self._bump()

    def set_root(self, layer: ShortcutLayer):
        self._layers = [layer]
        self._bump()

    def set_active_layer(self, layer: ShortcutLayer):
        if self.current_layer() is layer:
            return
        self._layers = [layer]
        self._bump()

    def push_layer(self, layer: ShortcutLayer):
        self._layers.append(layer)
        self._bump()

    def pop_layer(self):
        if len(self._layers) > 1:
            self._layers.pop()
            self._bump()

    def current_layer(self):
        if not self._layers:
            return None
        return self._layers[-1]

    def add_global(self, key, label, callback):
        k = _normalize_key(key)
        self._globals[k] = Shortcut(
            key=k,
            label=str(label),
            callback=callback,
        )
        self._bump()

    def hints(self, show_globals=True):
        shortcuts = []
        layer = self.current_layer()
        if layer:
            shortcuts.extend(layer.shortcuts.values())
        if show_globals:
            shortcuts.extend(self._globals.values())
        return shortcuts

    def visible_locals(self):
        layer = self.current_layer()
        if not layer:
            return []
        return list(layer.shortcuts.values())

    def visible_globals(self):
        return list(self._globals.values())
