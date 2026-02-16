import socket
import subprocess
import sys

from ..actions import Actions
from ..app import Component

DEFAULT_VIDEO_HOST = "127.0.0.1"
DEFAULT_VIDEO_PORT = 6502
DEFAULT_VIDEO_REFRESH_MS = 33
DEFAULT_VIDEO_ZOOM = 1
VIDEO_ERROR_DURATION_S = 5.0


class VideoPreviewManager(Component):
    def __init__(
        self,
        host=DEFAULT_VIDEO_HOST,
        port=DEFAULT_VIDEO_PORT,
        refresh_ms=DEFAULT_VIDEO_REFRESH_MS,
        zoom=DEFAULT_VIDEO_ZOOM,
        socket_path=None,
    ):
        super().__init__()
        self._host = host
        self._port = port
        self._refresh_ms = refresh_ms
        self._zoom = zoom
        self._socket_path = socket_path
        self._proc = None

    async def update(self):
        if self._proc and self._proc.poll() is not None:
            self._proc = None
        return False

    def toggle(self):
        if self._proc and self._proc.poll() is None:
            self._stop()
            return
        err = self._can_bind_udp()
        if err:
            self._set_error(f"Video preview bind failed: {err}")
            return
        self._start()

    def _set_error(self, message):
        self.app.dispatch_action(
            Actions.SET_UI_ERROR,
            (message, VIDEO_ERROR_DURATION_S),
        )

    def _can_bind_udp(self):
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        try:
            sock.bind((self._host, self._port))
        except OSError as exc:
            return exc
        finally:
            sock.close()
        return None

    def _start(self):
        args = [
            sys.executable,
            "-m",
            "py800mon.cli",
        ]
        if self._socket_path:
            args.extend(["--socket", self._socket_path])
        args.extend(
            [
                "video",
                "--host",
                self._host,
                "--port",
                str(self._port),
                "--refresh-ms",
                str(self._refresh_ms),
                "--zoom",
                str(self._zoom),
            ]
        )
        try:
            self._proc = subprocess.Popen(
                args,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
        except Exception as exc:
            self._proc = None
            self._set_error(f"Video preview failed: {exc}")

    def _stop(self):
        proc = self._proc
        if not proc:
            return
        if proc.poll() is None:
            proc.terminate()
            try:
                proc.wait(timeout=0.5)
            except Exception:
                proc.kill()
        self._proc = None
