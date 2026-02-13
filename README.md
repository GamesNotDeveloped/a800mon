# a800mon

![a800mon](assets/a800mon.png)

`a800mon` is a lightweight terminal-based monitor/debugger frontend for 8-bit emulators. It uses own binary protocol to communicate with the emulator.

Currently only Atari800 emulator is supported, and it's fork with Remote Monitor is required. Any emulator (or any other backend) can implement a protocol for which a800mon can be used.

Two versions of a800mon are supported for now:
- Python: `py800mon` – the base of the project
- Go: `go800mon` – experimental port to Go

## Required Emulator

This tool requires an Atari emulator, which implements Remote Monitor Binary Socket Protocol. For now, only forked Atari800 emuator implements it.

### Atari800 (fork)

Atari800 build/fork with the Remote Monitor feature compiled and enabled (RPC over UNIX socket).

Atari800 fork repository: https://github.com/a800mon/atari800

The fork provides Remote Monitor, which can be enabled by
`-remote-monitor` flag or `REMOTE_MONITOR=1` in the configuration file.

```bash
atari800 -remote-monitor
```

The default (and currently supported) transport is `socket`, set by
default. The default socket path is set to `/tmp/atari.sock`, which can
be changed bia `-remote-monitor-socket-path` flag or
`REMOTE_MONITOR_SOCKET_PATH` key in the configuration file.

## Install / Build

### Python (`py800mon`)

```bash
pip install --user -e .
```

### Go (`go800mon`)

```bash
go install ./go800mon/cmd/go800mon/
```

If `go800mon` is not visible in your shell, add `$(go env GOPATH)/bin` to `PATH` or set:

```bash
go env -w GOBIN="$HOME/.local/bin"
```

## Usage

Run the monitor:

```bash
py800mon
```

or

```bash
go800mon
```

### Command Line Options

Check for available options with:

```bash
py800mon --help
```

or

```bash
go800mon --help
```
