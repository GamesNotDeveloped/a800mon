import enum
import struct


class RpcException(Exception):
    pass


class ConnectionError(RpcException):
    pass


class TransportError(RpcException):
    pass


class InvalidTransportCommand(TransportError):
    pass


class CommandError(RpcException):
    pass


class Command(enum.Enum):
    PING = "ping"
    DLIST_ADDR = "dlist_addr"
    DLIST_DUMP = "dlist_dump"
    MEM_READ = "mem_read"
    MEM_READV = "mem_readv"
    CPU_STATE = "cpu_state"
    PAUSE = "pause"
    CONTINUE = "continue"
    STEP = "step"
    STEP_VBLANK = "step_vblank"
    STATUS = "status"
    RUN = "run"
    COLDSTART = "coldstart"
    WARMSTART = "warmstart"
    REMOVECARTRIGE = "removecartrige"
    STOP_EMULATOR = "stop_emulator"


class RpcClient:
    def __init__(self, transport):
        self._transport = transport
        self.last_error = None
        self._max_read = 0x400

    def call(self, command: Command, payload=None):
        try:
            internal_command = self._transport.translate_command(command)
        except KeyError:
            raise InvalidTransportCommand(
                f"Command not supported in the transport: {command}"
            )
        else:
            self.last_error = None

        try:
            status, data = self._transport.send(internal_command, payload=payload)
        except (ConnectionError, ConnectionResetError) as ex:
            self.last_error = ex
            raise ConnectionError(ex)
        else:
            self.last_error = None
        if status == 0:
            return data
        else:
            raise CommandError(status)

    def read_vector(self, addr: int):
        ptr = self.call(Command.MEM_READ, struct.pack("<HH", addr, 2))
        return ptr[0] | (ptr[1] << 8)

    def read_byte(self, addr: int) -> int:
        data = self.call(Command.MEM_READ, struct.pack("<HH", addr, 1))
        return data[0]

    def read_memory(self, addr: int, length: int):
        if length <= 0:
            return b""
        max_chunk = self._max_read
        if max_chunk <= 0 or length <= max_chunk:
            return self.call(Command.MEM_READ, struct.pack("<HH", addr, length))
        data = bytearray()
        remaining = length
        cur = addr
        while remaining:
            take = max_chunk if remaining > max_chunk else remaining
            data += self.call(Command.MEM_READ, struct.pack("<HH", cur, take))
            cur = (cur + take) & 0xFFFF
            remaining -= take
        return bytes(data)

    def read_memory_multiple(self, ranges):
        payload = struct.pack("<H", len(ranges)) + b"".join(
            struct.pack("<HH", addr, ln) for addr, ln in ranges
        )
        return self.call(Command.MEM_READV, payload)

    def read_display_list(self):
        return self.call(Command.DLIST_DUMP)

    def cpu_state(self):
        data = self.call(Command.CPU_STATE)
        return struct.unpack("<HHHBBBBB", data)
