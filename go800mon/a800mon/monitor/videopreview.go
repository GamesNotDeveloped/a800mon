package monitor

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const (
	defaultVideoHost       = "127.0.0.1"
	defaultVideoPort       = 6502
	defaultVideoRefreshMS  = 33
	defaultVideoZoom       = 1
	defaultVideoErrorDur   = 5 * time.Second
	defaultVideoKillWait   = 500 * time.Millisecond
)

type VideoPreviewManager struct {
	BaseComponent
	host      string
	port      int
	refreshMS int
	zoom      int
	socketPath string
	cmd       *exec.Cmd
	done      chan struct{}
}

func NewVideoPreviewManager(socketPath string) *VideoPreviewManager {
	return &VideoPreviewManager{
		host:      defaultVideoHost,
		port:      defaultVideoPort,
		refreshMS: defaultVideoRefreshMS,
		zoom:      defaultVideoZoom,
		socketPath: socketPath,
	}
}

func (v *VideoPreviewManager) Update(_ctx context.Context) (bool, error) {
	if v.cmd == nil || v.done == nil {
		return false, nil
	}
	select {
	case <-v.done:
		v.cmd = nil
		v.done = nil
	default:
	}
	return false, nil
}

func (v *VideoPreviewManager) Toggle() {
	if v.cmd != nil {
		v.stop()
		return
	}
	if err := v.canBindUDP(); err != nil {
		v.setError(fmt.Sprintf("Video preview bind failed: %v", err))
		return
	}
	v.start()
}

func (v *VideoPreviewManager) setError(message string) {
	v.App().DispatchAction(ActionSetUIError, UIError{
		Text:  message,
		Until: time.Now().Add(defaultVideoErrorDur),
	})
}

func (v *VideoPreviewManager) canBindUDP() error {
	addr := net.JoinHostPort(v.host, strconv.Itoa(v.port))
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

func (v *VideoPreviewManager) start() {
	exe, err := os.Executable()
	if err != nil {
		v.setError(fmt.Sprintf("Video preview failed: %v", err))
		return
	}
	args := []string{}
	if v.socketPath != "" {
		args = append(args, "--socket", v.socketPath)
	}
	args = append(
		args,
		"video",
		"--host", v.host,
		"--port", strconv.Itoa(v.port),
		"--refresh-ms", strconv.Itoa(v.refreshMS),
		"--zoom", strconv.Itoa(v.zoom),
	)
	cmd := exec.Command(exe, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		v.setError(fmt.Sprintf("Video preview failed: %v", err))
		return
	}
	done := make(chan struct{})
	v.cmd = cmd
	v.done = done
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
}

func (v *VideoPreviewManager) stop() {
	cmd := v.cmd
	if cmd == nil || cmd.Process == nil {
		v.cmd = nil
		v.done = nil
		return
	}
	_ = cmd.Process.Signal(os.Interrupt)
	if v.done != nil {
		select {
		case <-v.done:
		case <-time.After(defaultVideoKillWait):
			_ = cmd.Process.Kill()
			<-v.done
		}
	} else {
		_ = cmd.Process.Kill()
	}
	v.cmd = nil
	v.done = nil
}
