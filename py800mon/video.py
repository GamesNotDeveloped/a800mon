import asyncio
import queue
import signal
import socket
import struct
import threading
import time
import tkinter as tk

from .rpc import RpcClient
from .socket import SocketTransport

VIDEO_MAGIC = b"RMV1"
VIDEO_VERSION = 1
VIDEO_FORMAT_RGB888 = 1
VIDEO_HEADER_FMT = "<4sBBBBIHHHHHH"
VIDEO_HEADER_SIZE = struct.calcsize(VIDEO_HEADER_FMT)
KEYSPACE_HID = 1
CONSOL_DEFAULT = 0
WINDOW_TITLE = "a800mon video preview"
ERROR_TITLE_DURATION_S = 5.0
MOD_SHIFT = 1
MOD_CTRL = 2
MOD_ALT = 4
SHIFT_MASK = 0x0001
CTRL_MASK = 0x0004
ALT_MASK = 0x0008
HID_USAGE_A = 0x04
HID_USAGE_1 = 0x1E
HID_USAGE_0 = 0x27
HID_USAGE_ENTER = 0x28
HID_USAGE_ESCAPE = 0x29
HID_USAGE_BACKSPACE = 0x2A
HID_USAGE_TAB = 0x2B
HID_USAGE_SPACE = 0x2C
HID_USAGE_F1 = 0x3A
HID_USAGE_PRINT = 0x46
HID_USAGE_SCROLL_LOCK = 0x47
HID_USAGE_PAUSE = 0x48
HID_USAGE_INSERT = 0x49
HID_USAGE_HOME = 0x4A
HID_USAGE_PAGE_UP = 0x4B
HID_USAGE_DELETE = 0x4C
HID_USAGE_END = 0x4D
HID_USAGE_PAGE_DOWN = 0x4E
HID_USAGE_RIGHT = 0x4F
HID_USAGE_LEFT = 0x50
HID_USAGE_DOWN = 0x51
HID_USAGE_UP = 0x52
HID_USAGE_NUM_LOCK = 0x53
HID_USAGE_KP_DIVIDE = 0x54
HID_USAGE_KP_MULTIPLY = 0x55
HID_USAGE_KP_SUBTRACT = 0x56
HID_USAGE_KP_ADD = 0x57
HID_USAGE_KP_ENTER = 0x58
HID_USAGE_KP_1 = 0x59
HID_USAGE_KP_0 = 0x62
HID_USAGE_KP_DECIMAL = 0x63
HID_USAGE_KP_EQUAL = 0x67

HID_KEYSYM_MAP = {
    "return": HID_USAGE_ENTER,
    "kp_enter": HID_USAGE_KP_ENTER,
    "backspace": HID_USAGE_BACKSPACE,
    "tab": HID_USAGE_TAB,
    "escape": HID_USAGE_ESCAPE,
    "space": HID_USAGE_SPACE,
    "iso_left_tab": HID_USAGE_TAB,
    "left": HID_USAGE_LEFT,
    "right": HID_USAGE_RIGHT,
    "up": HID_USAGE_UP,
    "down": HID_USAGE_DOWN,
    "home": HID_USAGE_HOME,
    "end": HID_USAGE_END,
    "prior": HID_USAGE_PAGE_UP,
    "next": HID_USAGE_PAGE_DOWN,
    "insert": HID_USAGE_INSERT,
    "delete": HID_USAGE_DELETE,
    "print": HID_USAGE_PRINT,
    "printscreen": HID_USAGE_PRINT,
    "scroll_lock": HID_USAGE_SCROLL_LOCK,
    "pause": HID_USAGE_PAUSE,
    "num_lock": HID_USAGE_NUM_LOCK,
    "kp_divide": HID_USAGE_KP_DIVIDE,
    "kp_multiply": HID_USAGE_KP_MULTIPLY,
    "kp_subtract": HID_USAGE_KP_SUBTRACT,
    "kp_add": HID_USAGE_KP_ADD,
    "kp_equal": HID_USAGE_KP_EQUAL,
    "kp_decimal": HID_USAGE_KP_DECIMAL,
    "kp_0": HID_USAGE_KP_0,
    "kp_1": HID_USAGE_KP_1,
    "kp_2": HID_USAGE_KP_1 + 1,
    "kp_3": HID_USAGE_KP_1 + 2,
    "kp_4": HID_USAGE_KP_1 + 3,
    "kp_5": HID_USAGE_KP_1 + 4,
    "kp_6": HID_USAGE_KP_1 + 5,
    "kp_7": HID_USAGE_KP_1 + 6,
    "kp_8": HID_USAGE_KP_1 + 7,
    "kp_9": HID_USAGE_KP_1 + 8,
    "minus": 0x2D,
    "equal": 0x2E,
    "bracketleft": 0x2F,
    "bracketright": 0x30,
    "backslash": 0x31,
    "semicolon": 0x33,
    "apostrophe": 0x34,
    "grave": 0x35,
    "comma": 0x36,
    "period": 0x37,
    "slash": 0x38,
    "caps_lock": 0x39,
    "exclam": HID_USAGE_1,
    "at": HID_USAGE_1 + 1,
    "numbersign": HID_USAGE_1 + 2,
    "dollar": HID_USAGE_1 + 3,
    "percent": HID_USAGE_1 + 4,
    "asciicircum": HID_USAGE_1 + 5,
    "ampersand": HID_USAGE_1 + 6,
    "asterisk": HID_USAGE_1 + 7,
    "parenleft": HID_USAGE_1 + 8,
    "parenright": HID_USAGE_0,
    "underscore": 0x2D,
    "plus": 0x2E,
    "braceleft": 0x2F,
    "braceright": 0x30,
    "bar": 0x31,
    "colon": 0x33,
    "quotedbl": 0x34,
    "asciitilde": 0x35,
    "less": 0x36,
    "greater": 0x37,
    "question": 0x38,
    "!": HID_USAGE_1,
    "@": HID_USAGE_1 + 1,
    "#": HID_USAGE_1 + 2,
    "$": HID_USAGE_1 + 3,
    "%": HID_USAGE_1 + 4,
    "^": HID_USAGE_1 + 5,
    "&": HID_USAGE_1 + 6,
    "*": HID_USAGE_1 + 7,
    "(": HID_USAGE_1 + 8,
    ")": HID_USAGE_0,
    "_": 0x2D,
    "+": 0x2E,
    "{": 0x2F,
    "}": 0x30,
    "|": 0x31,
    ":": 0x33,
    '"': 0x34,
    "~": 0x35,
    "<": 0x36,
    ">": 0x37,
    "?": 0x38,
    "-": 0x2D,
    "=": 0x2E,
    "[": 0x2F,
    "]": 0x30,
    "\\": 0x31,
    ";": 0x33,
    "'": 0x34,
    "`": 0x35,
    ",": 0x36,
    ".": 0x37,
    "/": 0x38,
}

def _hid_usage_for_keysym(keysym):
    if not keysym:
        return None
    keysym = keysym.lower()
    usage = HID_KEYSYM_MAP.get(keysym)
    if usage is not None:
        return usage
    if keysym.startswith("f"):
        num = keysym[1:]
        if num.isdigit():
            idx = int(num)
            if 1 <= idx <= 12:
                return HID_USAGE_F1 + (idx - 1)
    if len(keysym) != 1:
        return None
    if "a" <= keysym <= "z":
        return HID_USAGE_A + (ord(keysym) - ord("a"))
    if "A" <= keysym <= "Z":
        return HID_USAGE_A + (ord(keysym) - ord("A"))
    if "1" <= keysym <= "9":
        return HID_USAGE_1 + (ord(keysym) - ord("1"))
    if keysym == "0":
        return HID_USAGE_0
    return HID_KEYSYM_MAP.get(keysym)


class VideoFrameAssembler:
    def __init__(self):
        self._lock = threading.Lock()
        self._frame_seq = None
        self._width = 0
        self._height = 0
        self._row_bytes = 0
        self._buffer = bytearray()
        self._updated = False

    def process(self, data):
        if len(data) < VIDEO_HEADER_SIZE:
            return

        (
            magic,
            version,
            fmt,
            _flags,
            _reserved,
            frame_seq,
            width,
            height,
            _x,
            y,
            rows,
            row_bytes,
        ) = struct.unpack_from(VIDEO_HEADER_FMT, data)

        if magic != VIDEO_MAGIC:
            return
        if version != VIDEO_VERSION or fmt != VIDEO_FORMAT_RGB888:
            return

        payload = data[VIDEO_HEADER_SIZE:]
        expected = rows * row_bytes
        if len(payload) != expected:
            return
        if y + rows > height:
            return

        with self._lock:
            if self._frame_seq is None or frame_seq > self._frame_seq:
                self._frame_seq = frame_seq
            if frame_seq != self._frame_seq:
                return

            if (
                width != self._width
                or height != self._height
                or row_bytes != self._row_bytes
                or len(self._buffer) != row_bytes * height
            ):
                self._width = width
                self._height = height
                self._row_bytes = row_bytes
                self._buffer = bytearray(row_bytes * height)

            start = y * row_bytes
            end = start + expected
            if end > len(self._buffer):
                return
            self._buffer[start:end] = payload
            self._updated = True

    def take_frame(self):
        with self._lock:
            if not self._updated or not self._buffer:
                return None
            self._updated = False
            return self._width, self._height, bytes(self._buffer)


def _load_pillow():
    try:
        from PIL import Image, ImageTk
    except Exception:
        raise SystemExit(
            "Pillow is required for video preview. Install with: pip install py800mon[video]"
        ) from None
    return Image, ImageTk


class VideoPreviewApp:
    def __init__(self, host, port, refresh_ms, zoom, socket_path):
        self._Image, self._ImageTk = _load_pillow()
        self._assembler = VideoFrameAssembler()
        self._sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self._sock.bind((host, port))
        self._sock.settimeout(0.2)
        self._stop = threading.Event()
        self._thread = threading.Thread(
            target=self._recv_loop, name="video-recv", daemon=True
        )
        self._socket_path = socket_path
        self._rpc_queue = queue.SimpleQueue() if socket_path else None
        self._rpc_thread = (
            threading.Thread(
                target=self._rpc_loop, name="video-rpc", daemon=True
            )
            if socket_path
            else None
        )
        self._rpc_error_queue = queue.SimpleQueue()
        self._pressed = {}

        self._root = tk.Tk()
        self._root.title(WINDOW_TITLE)
        self._label = tk.Label(self._root)
        self._label.pack(expand=True, fill="both")
        self._photo = None
        self._refresh_ms = refresh_ms
        self._zoom = zoom
        self._minsize_set = False
        self._nominal_width = None
        self._nominal_height = None
        self._min_width = None
        self._min_height = None
        self._last_state = None
        self._pending_restore = False
        self._last_restore = 0.0
        self._target_width = None
        self._target_height = None
        self._title_error = None
        self._title_error_until = 0.0
        self._root.protocol("WM_DELETE_WINDOW", self._on_close)
        signal.signal(signal.SIGINT, self._on_sigint)
        self._root.bind("<Configure>", self._on_resize)
        self._root.bind("<KeyPress>", self._on_key_press)
        self._root.bind("<KeyRelease>", self._on_key_release)

    def _recv_loop(self):
        while not self._stop.is_set():
            try:
                data, _addr = self._sock.recvfrom(2048)
            except socket.timeout:
                continue
            except OSError:
                break
            try:
                self._assembler.process(data)
            except Exception:
                pass

    def _rpc_loop(self):
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        rpc = RpcClient(SocketTransport(self._socket_path))
        while not self._stop.is_set():
            event = self._rpc_queue.get()
            if event is None:
                break
            action, keyspace, mods, consol, keycode = event
            try:
                loop.run_until_complete(
                    rpc.input_key(action, keyspace, mods, consol, keycode)
                )
            except Exception as exc:
                self._rpc_error_queue.put(f"RPC input failed: {exc}")
        loop.close()

    def _on_close(self):
        self._stop.set()
        try:
            self._sock.close()
        except OSError:
            pass
        if self._rpc_queue:
            self._rpc_queue.put(None)
        self._root.destroy()

    def _on_sigint(self, _sig, _frame):
        self._root.after(0, self._on_close)

    def _schedule_restore(self):
        now = time.monotonic()
        if self._pending_restore or (now - self._last_restore) < 0.5:
            return
        self._pending_restore = True
        self._last_restore = now
        self._root.after_idle(self._restore_nominal)

    def _restore_nominal(self):
        self._pending_restore = False
        if not self._nominal_width or not self._nominal_height:
            return
        self._root.geometry(f"{self._nominal_width}x{self._nominal_height}")
        self._target_width = self._nominal_width
        self._target_height = self._nominal_height

    def _on_resize(self, event):
        if event.widget is self._root:
            width = event.width
            height = event.height
            if self._minsize_set and self._min_width and self._min_height:
                if width < self._min_width or height < self._min_height:
                    self._schedule_restore()
                width = max(width, self._min_width)
                height = max(height, self._min_height)

            self._target_width = width
            self._target_height = height
            current_state = self._root.state()
            if (
                self._nominal_width
                and self._nominal_height
                and self._last_state == "zoomed"
                and current_state == "normal"
            ):
                self._root.geometry(
                    f"{self._nominal_width}x{self._nominal_height}"
                )
                self._target_width = self._nominal_width
                self._target_height = self._nominal_height
            self._last_state = current_state

    def _event_mods(self, event):
        mods = 0
        if event.state & SHIFT_MASK:
            mods |= MOD_SHIFT
        if event.state & CTRL_MASK:
            mods |= MOD_CTRL
        if event.state & ALT_MASK:
            mods |= MOD_ALT
        return mods

    def _event_keycode(self, event):
        return _hid_usage_for_keysym(event.keysym)

    def _send_key(self, action, keycode, mods):
        if not self._rpc_queue:
            return
        self._rpc_queue.put(
            (action, KEYSPACE_HID, mods, CONSOL_DEFAULT, keycode)
        )

    def _on_key_press(self, event):
        if event.keysym == "F12":
            self._on_close()
            return "break"
        keycode = self._event_keycode(event)
        if keycode is None:
            return None
        mods = self._event_mods(event)
        self._pressed[event.keysym] = (keycode, mods)
        self._send_key(1, keycode, mods)
        return None

    def _on_key_release(self, event):
        if event.keysym == "F12":
            return "break"
        info = self._pressed.pop(event.keysym, None)
        if info:
            keycode, mods = info
            self._send_key(0, keycode, mods)
            return None
        keycode = self._event_keycode(event)
        if keycode is None:
            return None
        mods = self._event_mods(event)
        self._send_key(0, keycode, mods)
        return None

    def _update_frame(self):
        try:
            self._update_title_error()
            frame = self._assembler.take_frame()
            if frame:
                width, height, pixels = frame
                if not self._minsize_set:
                    min_w = width
                    min_h = height
                    nominal_w = width * self._zoom
                    nominal_h = height * self._zoom
                    self._min_width = min_w
                    self._min_height = min_h
                    self._nominal_width = nominal_w
                    self._nominal_height = nominal_h
                    self._root.minsize(min_w, min_h)
                    self._root.geometry(f"{nominal_w}x{nominal_h}")
                    self._target_width = nominal_w
                    self._target_height = nominal_h
                    self._minsize_set = True

                image = self._Image.frombytes("RGB", (width, height), pixels)
                target_w = self._target_width or width
                target_h = self._target_height or height
                if target_w < 1 or target_h < 1:
                    target_w = width
                    target_h = height

                scale = min(target_w / width, target_h / height)
                new_w = max(1, int(width * scale))
                new_h = max(1, int(height * scale))
                if new_w != width or new_h != height:
                    image = image.resize(
                        (new_w, new_h), resample=self._Image.NEAREST
                    )
                if new_w != target_w or new_h != target_h:
                    padded = self._Image.new("RGB", (target_w, target_h), (0, 0, 0))
                    left = (target_w - new_w) // 2
                    top = (target_h - new_h) // 2
                    padded.paste(image, (left, top))
                    image = padded

                self._photo = self._ImageTk.PhotoImage(image, master=self._root)
                self._label.configure(image=self._photo)
        except Exception:
            pass
        self._root.after(self._refresh_ms, self._update_frame)

    def _update_title_error(self):
        message = None
        while True:
            try:
                message = self._rpc_error_queue.get_nowait()
            except queue.Empty:
                break
        if message:
            message = message.strip()
            if message and message != self._title_error:
                self._title_error = message
                self._title_error_until = (
                    time.monotonic() + ERROR_TITLE_DURATION_S
                )
                self._root.title(f"{WINDOW_TITLE} - {message}")
        if self._title_error and time.monotonic() >= self._title_error_until:
            self._title_error = None
            self._title_error_until = 0.0
            self._root.title(WINDOW_TITLE)

    def run(self):
        self._thread.start()
        if self._rpc_thread:
            self._rpc_thread.start()
        self._update_frame()
        self._root.focus_set()
        try:
            self._root.mainloop()
        except KeyboardInterrupt:
            self._on_close()


def run_video_preview(
    host="127.0.0.1",
    port=6502,
    refresh_ms=33,
    zoom=1,
    socket_path=None,
):
    VideoPreviewApp(
        host, port, refresh_ms, zoom=zoom, socket_path=socket_path
    ).run()
