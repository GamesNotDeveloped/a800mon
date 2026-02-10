package rpc

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Command byte

const (
	CmdPing            Command = 1
	CmdDlistAddr       Command = 2
	CmdMemRead         Command = 3
	CmdDlistDump       Command = 4
	CmdCPUState        Command = 5
	CmdPause           Command = 6
	CmdContinue        Command = 7
	CmdStep            Command = 8
	CmdStepVBlank      Command = 9
	CmdStatus          Command = 10
	CmdMemReadV        Command = 11
	CmdRun             Command = 12
	CmdColdstart       Command = 13
	CmdWarmstart       Command = 14
	CmdRemoveCartrige  Command = 15
	CmdStopEmulator    Command = 16
	CmdRestartEmulator Command = 28
	CmdRemoveTape      Command = 17
	CmdRemoveDisks     Command = 18
	CmdHistory         Command = 19
	CmdBuiltinMonitor  Command = 20
	CmdWriteMemory     Command = 21
	CmdBPClear         Command = 22
	CmdBPAddClause     Command = 23
	CmdBPDeleteClause  Command = 24
	CmdBPSetEnabled    Command = 25
	CmdBPList          Command = 26
	CmdConfig          Command = 27
)

type Status struct {
	Paused   bool
	EmuMS    uint64
	ResetMS  uint64
	Crashed  bool
	StateSeq uint64
}

type CPUState struct {
	YPos uint16
	XPos uint16
	PC   uint16
	A    byte
	X    byte
	Y    byte
	S    byte
	P    byte
}

type HistoryEntry struct {
	Y   byte
	X   byte
	PC  uint16
	Op0 byte
	Op1 byte
	Op2 byte
}

type BreakpointCondition struct {
	Type  byte
	Op    byte
	Addr  uint16
	Value uint16
}

type BreakpointList struct {
	Enabled bool
	Clauses [][]BreakpointCondition
}

func (e HistoryEntry) OpBytes() []byte {
	return []byte{e.Op0, e.Op1, e.Op2}
}

type CPUStateStringer interface {
	FormatCPU(CPUState) string
}

type CommandError struct {
	Status byte
	Data   []byte
}

func (e CommandError) Error() string {
	if len(e.Data) == 0 {
		return fmt.Sprintf("remote command error: status=%d", e.Status)
	}
	return fmt.Sprintf("remote command error: status=%d msg=%s", e.Status, strings.TrimSpace(string(e.Data)))
}

type Client struct {
	path    string
	timeout time.Duration

	mu         sync.Mutex
	conn       net.Conn
	lastError  error
	configCaps []uint16
}

func New(path string) *Client {
	return &Client{
		path:    path,
		timeout: 500 * time.Millisecond,
	}
}

func (c *Client) Path() string {
	return c.path
}

func (c *Client) SetTimeout(timeout time.Duration) {
	if timeout <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timeout = timeout
}

func (c *Client) LastError() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastError
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.disconnectLocked()
}

func (c *Client) disconnectLocked() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.configCaps = nil
}

func (c *Client) ensureConnectedLocked(ctx context.Context) error {
	if c.conn != nil {
		return nil
	}
	dialer := net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "unix", c.path)
	if err != nil {
		return formatConnectError(c.path, err)
	}
	c.conn = conn
	c.readConfigOnConnectLocked(ctx)
	if c.conn == nil {
		if c.lastError != nil {
			return c.lastError
		}
		return errors.New("socket connection lost during CONFIG handshake")
	}
	return nil
}

func (c *Client) readConfigOnConnectLocked(ctx context.Context) {
	if c.conn == nil {
		c.configCaps = nil
		return
	}
	c.setDeadlineLocked(ctx)
	packet := []byte{byte(CmdConfig), 0, 0}
	if _, err := c.conn.Write(packet); err != nil {
		c.disconnectLocked()
		c.lastError = err
		return
	}
	hdr := make([]byte, 3)
	if _, err := io.ReadFull(c.conn, hdr); err != nil {
		c.disconnectLocked()
		c.lastError = err
		return
	}
	status := hdr[0]
	ln := int(binary.LittleEndian.Uint16(hdr[1:3]))
	var data []byte
	if ln > 0 {
		data = make([]byte, ln)
		if _, err := io.ReadFull(c.conn, data); err != nil {
			c.disconnectLocked()
			c.lastError = err
			return
		}
	}
	if status != 0 {
		c.configCaps = nil
		return
	}
	caps, err := parseConfigPayload(data)
	if err != nil {
		c.configCaps = nil
		return
	}
	c.configCaps = caps
}

func formatConnectError(path string, err error) error {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		msg := errno.Error()
		if msg != "" {
			msg = strings.ToUpper(msg[:1]) + msg[1:]
		}
		return fmt.Errorf("Cannot connect to socket %s: [Errno %d] %s", path, int(errno), msg)
	}
	return fmt.Errorf("Cannot connect to socket %s: %s", path, err)
}

func (c *Client) setDeadlineLocked(ctx context.Context) {
	if c.conn == nil {
		return
	}
	deadline := time.Now().Add(c.timeout)
	if ctx != nil {
		if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
			deadline = dl
		}
	}
	_ = c.conn.SetDeadline(deadline)
}

func (c *Client) Call(ctx context.Context, command Command, payload []byte) ([]byte, error) {
	if len(payload) > 0xFFFF {
		return nil, fmt.Errorf("payload too large: %d", len(payload))
	}
	if ctx == nil {
		ctx = context.Background()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnectedLocked(ctx); err != nil {
		c.lastError = err
		return nil, err
	}
	if c.conn == nil {
		err := errors.New("socket not connected")
		c.lastError = err
		return nil, err
	}
	c.setDeadlineLocked(ctx)

	packet := make([]byte, 3+len(payload))
	packet[0] = byte(command)
	binary.LittleEndian.PutUint16(packet[1:3], uint16(len(payload)))
	copy(packet[3:], payload)

	if _, err := c.conn.Write(packet); err != nil {
		c.disconnectLocked()
		c.lastError = err
		return nil, err
	}

	hdr := make([]byte, 3)
	if _, err := io.ReadFull(c.conn, hdr); err != nil {
		c.disconnectLocked()
		c.lastError = err
		return nil, err
	}
	status := hdr[0]
	ln := int(binary.LittleEndian.Uint16(hdr[1:3]))
	var data []byte
	if ln > 0 {
		data = make([]byte, ln)
		if _, err := io.ReadFull(c.conn, data); err != nil {
			c.disconnectLocked()
			c.lastError = err
			return nil, err
		}
	}
	if status != 0 {
		err := CommandError{
			Status: status,
			Data:   append([]byte(nil), data...),
		}
		c.lastError = err
		return nil, err
	}
	if ln == 0 {
		c.lastError = nil
		return nil, nil
	}
	c.lastError = nil
	return data, nil
}

func (c *Client) ReadVector(ctx context.Context, addr uint16) (uint16, error) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint16(buf[0:2], addr)
	binary.LittleEndian.PutUint16(buf[2:4], 2)
	data, err := c.Call(ctx, CmdMemRead, buf)
	if err != nil {
		return 0, err
	}
	if len(data) < 2 {
		return 0, errors.New("mem_read vector payload too short")
	}
	return uint16(data[0]) | (uint16(data[1]) << 8), nil
}

func (c *Client) ReadByte(ctx context.Context, addr uint16) (byte, error) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint16(buf[0:2], addr)
	binary.LittleEndian.PutUint16(buf[2:4], 1)
	data, err := c.Call(ctx, CmdMemRead, buf)
	if err != nil {
		return 0, err
	}
	if len(data) < 1 {
		return 0, errors.New("mem_read byte payload too short")
	}
	return data[0], nil
}

func (c *Client) ReadMemory(ctx context.Context, addr, length uint16) ([]byte, error) {
	if length == 0 {
		return nil, nil
	}
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint16(buf[0:2], addr)
	binary.LittleEndian.PutUint16(buf[2:4], length)
	return c.Call(ctx, CmdMemRead, buf)
}

func (c *Client) WriteMemory(ctx context.Context, addr uint16, data []byte) error {
	if len(data) > 0xFFFF {
		return fmt.Errorf("write_memory payload too long: %d bytes (max 65535)", len(data))
	}
	payload := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint16(payload[0:2], addr)
	binary.LittleEndian.PutUint16(payload[2:4], uint16(len(data)))
	copy(payload[4:], data)
	_, err := c.Call(ctx, CmdWriteMemory, payload)
	return err
}

func (c *Client) ReadMemoryChunked(ctx context.Context, addr uint16, length int, maxChunk int) ([]byte, error) {
	if length <= 0 {
		return nil, nil
	}
	if maxChunk <= 0 {
		maxChunk = 0x400
	}
	if length <= maxChunk {
		return c.ReadMemory(ctx, addr, uint16(length))
	}
	res := make([]byte, 0, length)
	remaining := length
	cur := addr
	for remaining > 0 {
		take := remaining
		if take > maxChunk {
			take = maxChunk
		}
		chunk, err := c.ReadMemory(ctx, cur, uint16(take))
		if err != nil {
			return nil, err
		}
		res = append(res, chunk...)
		cur = uint16((int(cur) + take) & 0xFFFF)
		remaining -= take
	}
	return res, nil
}

func (c *Client) ReadDisplayList(ctx context.Context) ([]byte, error) {
	return c.Call(ctx, CmdDlistDump, nil)
}

func (c *Client) ReadDisplayListAt(ctx context.Context, startAddr uint16) ([]byte, error) {
	payload := make([]byte, 2)
	binary.LittleEndian.PutUint16(payload, startAddr)
	return c.Call(ctx, CmdDlistDump, payload)
}

func (c *Client) Status(ctx context.Context) (Status, error) {
	data, err := c.Call(ctx, CmdStatus, nil)
	if err != nil {
		return Status{}, err
	}
	if len(data) < 21 {
		return Status{}, errors.New("status payload too short")
	}
	pausedByte := data[0]
	return Status{
		Paused:   pausedByte&0x01 != 0,
		Crashed:  pausedByte&0x80 != 0,
		EmuMS:    binary.LittleEndian.Uint64(data[1:9]),
		ResetMS:  binary.LittleEndian.Uint64(data[9:17]),
		StateSeq: uint64(binary.LittleEndian.Uint32(data[17:21])),
	}, nil
}

func (c *Client) CPUState(ctx context.Context) (CPUState, error) {
	data, err := c.Call(ctx, CmdCPUState, nil)
	if err != nil {
		return CPUState{}, err
	}
	if len(data) < 11 {
		return CPUState{}, errors.New("cpu_state payload too short")
	}
	off := 0
	return CPUState{
		YPos: binary.LittleEndian.Uint16(data[off : off+2]),
		XPos: binary.LittleEndian.Uint16(data[off+2 : off+4]),
		PC:   binary.LittleEndian.Uint16(data[off+4 : off+6]),
		A:    data[off+6],
		X:    data[off+7],
		Y:    data[off+8],
		S:    data[off+9],
		P:    data[off+10],
	}, nil
}

func (c *Client) History(ctx context.Context) ([]HistoryEntry, error) {
	data, err := c.Call(ctx, CmdHistory, nil)
	if err != nil {
		return nil, err
	}
	if len(data) < 1 {
		return nil, errors.New("history payload too short")
	}
	count := int(data[0])
	expected := 1 + count*7
	if len(data) < expected {
		return nil, fmt.Errorf("history payload too short: got=%d expected=%d", len(data), expected)
	}
	entries := make([]HistoryEntry, 0, count)
	off := 1
	for i := 0; i < count; i++ {
		entries = append(entries, HistoryEntry{
			Y:   data[off],
			X:   data[off+1],
			PC:  binary.LittleEndian.Uint16(data[off+2 : off+4]),
			Op0: data[off+4],
			Op1: data[off+5],
			Op2: data[off+6],
		})
		off += 7
	}
	return entries, nil
}

func (c *Client) Config(ctx context.Context) ([]uint16, error) {
	data, err := c.Call(ctx, CmdConfig, nil)
	if err != nil {
		return nil, err
	}
	caps, err := parseConfigPayload(data)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.configCaps = append([]uint16(nil), caps...)
	c.mu.Unlock()
	return caps, nil
}

func (c *Client) BPClear(ctx context.Context) error {
	_, err := c.Call(ctx, CmdBPClear, nil)
	return err
}

func (c *Client) BPAddClause(ctx context.Context, conditions []BreakpointCondition) (uint16, error) {
	if len(conditions) == 0 {
		return 0, errors.New("breakpoint clause must have at least one condition")
	}
	if len(conditions) > 20 {
		return 0, errors.New("breakpoint clause exceeds maximum of 20 conditions")
	}
	payload := make([]byte, 4+len(conditions)*6)
	binary.LittleEndian.PutUint16(payload[0:2], 0xFFFF)
	payload[2] = byte(len(conditions))
	payload[3] = 0
	offset := 4
	for _, cond := range conditions {
		payload[offset] = cond.Type
		payload[offset+1] = cond.Op
		binary.LittleEndian.PutUint16(payload[offset+2:offset+4], cond.Addr)
		binary.LittleEndian.PutUint16(payload[offset+4:offset+6], cond.Value)
		offset += 6
	}
	data, err := c.Call(ctx, CmdBPAddClause, payload)
	if err != nil {
		return 0, err
	}
	if len(data) < 2 {
		return 0, errors.New("bp_add_clause payload too short")
	}
	return binary.LittleEndian.Uint16(data[:2]), nil
}

func (c *Client) BPDeleteClause(ctx context.Context, clauseIndex uint16) error {
	payload := make([]byte, 2)
	binary.LittleEndian.PutUint16(payload, clauseIndex)
	_, err := c.Call(ctx, CmdBPDeleteClause, payload)
	return err
}

func (c *Client) BPSetEnabled(ctx context.Context, enabled bool) (bool, error) {
	value := byte(0)
	if enabled {
		value = 1
	}
	data, err := c.Call(ctx, CmdBPSetEnabled, []byte{value})
	if err != nil {
		return false, err
	}
	if len(data) < 1 {
		return false, errors.New("bp_set_enabled payload too short")
	}
	return data[0] != 0, nil
}

func (c *Client) BPList(ctx context.Context) (BreakpointList, error) {
	data, err := c.Call(ctx, CmdBPList, nil)
	if err != nil {
		return BreakpointList{}, err
	}
	if len(data) < 3 {
		return BreakpointList{}, errors.New("bp_list payload too short")
	}
	out := BreakpointList{
		Enabled: data[0] != 0,
	}
	clauseCount := int(binary.LittleEndian.Uint16(data[1:3]))
	offset := 3
	out.Clauses = make([][]BreakpointCondition, 0, clauseCount)
	for i := 0; i < clauseCount; i++ {
		if offset+2 > len(data) {
			return BreakpointList{}, errors.New("bp_list payload too short (clause header)")
		}
		condCount := int(data[offset])
		offset += 2 // cond_count + reserved
		clause := make([]BreakpointCondition, 0, condCount)
		for j := 0; j < condCount; j++ {
			if offset+6 > len(data) {
				return BreakpointList{}, errors.New("bp_list payload too short (condition)")
			}
			clause = append(clause, BreakpointCondition{
				Type:  data[offset],
				Op:    data[offset+1],
				Addr:  binary.LittleEndian.Uint16(data[offset+2 : offset+4]),
				Value: binary.LittleEndian.Uint16(data[offset+4 : offset+6]),
			})
			offset += 6
		}
		out.Clauses = append(out.Clauses, clause)
	}
	return out, nil
}

func parseConfigPayload(data []byte) ([]uint16, error) {
	if len(data) < 2 {
		return nil, errors.New("config payload too short")
	}
	count := int(binary.LittleEndian.Uint16(data[:2]))
	expected := 2 + count*2
	if len(data) < expected {
		return nil, fmt.Errorf("config payload too short: got=%d expected=%d", len(data), expected)
	}
	caps := make([]uint16, 0, count)
	offset := 2
	for i := 0; i < count; i++ {
		caps = append(caps, binary.LittleEndian.Uint16(data[offset:offset+2]))
		offset += 2
	}
	return caps, nil
}
