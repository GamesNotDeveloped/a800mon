package a800mon

import (
	"context"
	"encoding/binary"
	"image/color"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	videoMagic           = "RMV1"
	videoVersion         = 1
	videoFormatRGB888    = 1
	videoHeaderSize      = 24
	videoDefaultWidth    = 384
	videoDefaultHeight   = 240
	videoReadBufferBytes = 2048
	videoKeyspaceHID     = 1
	videoConsolDefault   = 0
	videoWindowTitle     = "a800mon video preview"
	hidUsageA            = 0x04
	hidUsage1            = 0x1E
	hidUsage0            = 0x27
	hidUsageEnter        = 0x28
	hidUsageEscape       = 0x29
	hidUsageBackspace    = 0x2A
	hidUsageTab          = 0x2B
	hidUsageSpace        = 0x2C
	hidUsageMinus        = 0x2D
	hidUsageEqual        = 0x2E
	hidUsageBracketLeft  = 0x2F
	hidUsageBracketRight = 0x30
	hidUsageBackslash    = 0x31
	hidUsageSemicolon    = 0x33
	hidUsageQuote        = 0x34
	hidUsageGrave        = 0x35
	hidUsageComma        = 0x36
	hidUsagePeriod       = 0x37
	hidUsageSlash        = 0x38
	hidUsageCapsLock     = 0x39
	hidUsageF1           = 0x3A
	hidUsagePrint        = 0x46
	hidUsageScrollLock   = 0x47
	hidUsagePause        = 0x48
	hidUsageInsert       = 0x49
	hidUsageHome         = 0x4A
	hidUsagePageUp       = 0x4B
	hidUsageDelete       = 0x4C
	hidUsageEnd          = 0x4D
	hidUsagePageDown     = 0x4E
	hidUsageRight        = 0x4F
	hidUsageLeft         = 0x50
	hidUsageDown         = 0x51
	hidUsageUp           = 0x52
	hidUsageNumLock      = 0x53
	hidUsageKPDivide     = 0x54
	hidUsageKPMultiply   = 0x55
	hidUsageKPSubtract   = 0x56
	hidUsageKPAdd        = 0x57
	hidUsageKPEnter      = 0x58
	hidUsageKP1          = 0x59
	hidUsageKP0          = 0x62
	hidUsageKPDecimal    = 0x63
	hidUsageKPEqual      = 0x67
)

type videoFrameAssembler struct {
	mu       sync.Mutex
	frameSeq uint32
	hasFrame bool
	width    int
	height   int
	rowBytes int
	buffer   []byte
	updated  bool
}

func (a *videoFrameAssembler) process(data []byte) {
	if len(data) < videoHeaderSize {
		return
	}
	if string(data[:4]) != videoMagic {
		return
	}
	if data[4] != videoVersion || data[5] != videoFormatRGB888 {
		return
	}
	frameSeq := binary.LittleEndian.Uint32(data[8:12])
	width := int(binary.LittleEndian.Uint16(data[12:14]))
	height := int(binary.LittleEndian.Uint16(data[14:16]))
	y := int(binary.LittleEndian.Uint16(data[18:20]))
	rows := int(binary.LittleEndian.Uint16(data[20:22]))
	rowBytes := int(binary.LittleEndian.Uint16(data[22:24]))
	if width < 1 || height < 1 || rows < 1 || rowBytes < 1 {
		return
	}
	payload := data[videoHeaderSize:]
	expected := rows * rowBytes
	if len(payload) != expected {
		return
	}
	if y+rows > height {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.hasFrame || frameSeq > a.frameSeq {
		a.frameSeq = frameSeq
		a.hasFrame = true
	}
	if frameSeq != a.frameSeq {
		return
	}
	if width != a.width || height != a.height || rowBytes != a.rowBytes || len(a.buffer) != rowBytes*height {
		a.width = width
		a.height = height
		a.rowBytes = rowBytes
		a.buffer = make([]byte, rowBytes*height)
	}
	start := y * rowBytes
	end := start + expected
	if end > len(a.buffer) {
		return
	}
	copy(a.buffer[start:end], payload)
	a.updated = true
}

func (a *videoFrameAssembler) takeFrame() (int, int, []byte, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.updated || len(a.buffer) == 0 {
		return 0, 0, nil, false
	}
	a.updated = false
	out := make([]byte, len(a.buffer))
	copy(out, a.buffer)
	return a.width, a.height, out, true
}

type inputEvent struct {
	action   byte
	keyspace byte
	mods     byte
	consol   byte
	keycode  uint16
}

type keyState struct {
	keycode uint16
	mods    byte
}

type videoPreview struct {
	assembler  *videoFrameAssembler
	width      int
	height     int
	zoom       int
	image      *ebiten.Image
	rgba       []byte
	outsideW   int
	outsideH   int
	minW       int
	minH       int
	nominalW   int
	nominalH   int
	lastResize time.Time
	inputCh    chan inputEvent
	errCh      chan string
	keyStates  map[ebiten.Key]keyState
	keys       []ebiten.Key
	lastError  string
	errorUntil time.Time
}

func newVideoPreview(assembler *videoFrameAssembler, zoom int, inputCh chan inputEvent, errCh chan string) *videoPreview {
	return &videoPreview{
		assembler: assembler,
		zoom:      zoom,
		width:     videoDefaultWidth,
		height:    videoDefaultHeight,
		minW:      videoDefaultWidth,
		minH:      videoDefaultHeight,
		nominalW:  videoDefaultWidth * zoom,
		nominalH:  videoDefaultHeight * zoom,
		inputCh:   inputCh,
		errCh:     errCh,
		keyStates: map[ebiten.Key]keyState{},
		keys:      defaultVideoKeys(),
	}
}

func defaultVideoKeys() []ebiten.Key {
	keys := make([]ebiten.Key, 0, 96)
	for key := ebiten.KeyA; key <= ebiten.KeyZ; key++ {
		keys = append(keys, key)
	}
	for key := ebiten.KeyDigit0; key <= ebiten.KeyDigit9; key++ {
		keys = append(keys, key)
	}
	for key := ebiten.KeyF1; key <= ebiten.KeyF12; key++ {
		keys = append(keys, key)
	}
	for key := ebiten.KeyNumpad0; key <= ebiten.KeyNumpad9; key++ {
		keys = append(keys, key)
	}
	keys = append(
		keys,
		ebiten.KeyEnter,
		ebiten.KeyTab,
		ebiten.KeyBackspace,
		ebiten.KeyEscape,
		ebiten.KeySpace,
		ebiten.KeyMinus,
		ebiten.KeyEqual,
		ebiten.KeyBracketLeft,
		ebiten.KeyBracketRight,
		ebiten.KeyBackslash,
		ebiten.KeySemicolon,
		ebiten.KeyQuote,
		ebiten.KeyBackquote,
		ebiten.KeyComma,
		ebiten.KeyPeriod,
		ebiten.KeySlash,
		ebiten.KeyCapsLock,
		ebiten.KeyArrowLeft,
		ebiten.KeyArrowRight,
		ebiten.KeyArrowUp,
		ebiten.KeyArrowDown,
		ebiten.KeyHome,
		ebiten.KeyEnd,
		ebiten.KeyPageUp,
		ebiten.KeyPageDown,
		ebiten.KeyInsert,
		ebiten.KeyDelete,
		ebiten.KeyPrintScreen,
		ebiten.KeyScrollLock,
		ebiten.KeyPause,
		ebiten.KeyNumLock,
		ebiten.KeyNumpadDivide,
		ebiten.KeyNumpadMultiply,
		ebiten.KeyNumpadSubtract,
		ebiten.KeyNumpadAdd,
		ebiten.KeyNumpadEnter,
		ebiten.KeyNumpadDecimal,
		ebiten.KeyNumpadEqual,
	)
	return keys
}

func currentInputMods() byte {
	mods := byte(0)
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		mods |= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		mods |= 2
	}
	if ebiten.IsKeyPressed(ebiten.KeyAlt) {
		mods |= 4
	}
	return mods
}

func keycodeForKey(key ebiten.Key) (uint16, bool) {
	if key >= ebiten.KeyA && key <= ebiten.KeyZ {
		return uint16(hidUsageA + int(key-ebiten.KeyA)), true
	}
	if key >= ebiten.KeyDigit1 && key <= ebiten.KeyDigit9 {
		return uint16(hidUsage1 + int(key-ebiten.KeyDigit1)), true
	}
	if key == ebiten.KeyDigit0 {
		return hidUsage0, true
	}
	if key >= ebiten.KeyF1 && key <= ebiten.KeyF12 {
		return uint16(hidUsageF1 + int(key-ebiten.KeyF1)), true
	}
	if key >= ebiten.KeyNumpad1 && key <= ebiten.KeyNumpad9 {
		return uint16(hidUsageKP1 + int(key-ebiten.KeyNumpad1)), true
	}
	if key == ebiten.KeyNumpad0 {
		return hidUsageKP0, true
	}
	switch key {
	case ebiten.KeyEnter:
		return hidUsageEnter, true
	case ebiten.KeyTab:
		return hidUsageTab, true
	case ebiten.KeyBackspace:
		return hidUsageBackspace, true
	case ebiten.KeyEscape:
		return hidUsageEscape, true
	case ebiten.KeySpace:
		return hidUsageSpace, true
	case ebiten.KeyMinus:
		return hidUsageMinus, true
	case ebiten.KeyEqual:
		return hidUsageEqual, true
	case ebiten.KeyBracketLeft:
		return hidUsageBracketLeft, true
	case ebiten.KeyBracketRight:
		return hidUsageBracketRight, true
	case ebiten.KeyBackslash:
		return hidUsageBackslash, true
	case ebiten.KeySemicolon:
		return hidUsageSemicolon, true
	case ebiten.KeyQuote:
		return hidUsageQuote, true
	case ebiten.KeyBackquote:
		return hidUsageGrave, true
	case ebiten.KeyComma:
		return hidUsageComma, true
	case ebiten.KeyPeriod:
		return hidUsagePeriod, true
	case ebiten.KeySlash:
		return hidUsageSlash, true
	case ebiten.KeyCapsLock:
		return hidUsageCapsLock, true
	case ebiten.KeyArrowRight:
		return hidUsageRight, true
	case ebiten.KeyArrowLeft:
		return hidUsageLeft, true
	case ebiten.KeyArrowDown:
		return hidUsageDown, true
	case ebiten.KeyArrowUp:
		return hidUsageUp, true
	case ebiten.KeyHome:
		return hidUsageHome, true
	case ebiten.KeyEnd:
		return hidUsageEnd, true
	case ebiten.KeyPageUp:
		return hidUsagePageUp, true
	case ebiten.KeyPageDown:
		return hidUsagePageDown, true
	case ebiten.KeyInsert:
		return hidUsageInsert, true
	case ebiten.KeyDelete:
		return hidUsageDelete, true
	case ebiten.KeyPrintScreen:
		return hidUsagePrint, true
	case ebiten.KeyScrollLock:
		return hidUsageScrollLock, true
	case ebiten.KeyPause:
		return hidUsagePause, true
	case ebiten.KeyNumLock:
		return hidUsageNumLock, true
	case ebiten.KeyNumpadDivide:
		return hidUsageKPDivide, true
	case ebiten.KeyNumpadMultiply:
		return hidUsageKPMultiply, true
	case ebiten.KeyNumpadSubtract:
		return hidUsageKPSubtract, true
	case ebiten.KeyNumpadAdd:
		return hidUsageKPAdd, true
	case ebiten.KeyNumpadEnter:
		return hidUsageKPEnter, true
	case ebiten.KeyNumpadDecimal:
		return hidUsageKPDecimal, true
	case ebiten.KeyNumpadEqual:
		return hidUsageKPEqual, true
	}
	return 0, false
}

func (v *videoPreview) sendKey(action byte, keycode uint16, mods byte) {
	if v.inputCh == nil {
		return
	}
	v.inputCh <- inputEvent{
		action:   action,
		keyspace: videoKeyspaceHID,
		mods:     mods,
		consol:   videoConsolDefault,
		keycode:  keycode,
	}
}

func (v *videoPreview) handleInput() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		return ebiten.Termination
	}
	mods := currentInputMods()
	for _, key := range v.keys {
		if inpututil.IsKeyJustPressed(key) {
			keycode, ok := keycodeForKey(key)
			if !ok {
				continue
			}
			v.keyStates[key] = keyState{keycode: keycode, mods: mods}
			v.sendKey(1, keycode, mods)
		}
		if inpututil.IsKeyJustReleased(key) {
			state, ok := v.keyStates[key]
			if ok {
				v.sendKey(0, state.keycode, state.mods)
				delete(v.keyStates, key)
				continue
			}
			keycode, ok := keycodeForKey(key)
			if !ok {
				continue
			}
			v.sendKey(0, keycode, mods)
		}
	}
	return nil
}

func (v *videoPreview) handleErrors() {
	if v.errCh == nil {
		return
	}
	for {
		select {
		case msg := <-v.errCh:
			if msg == "" {
				continue
			}
			if msg != v.lastError || time.Now().After(v.errorUntil) {
				v.lastError = msg
				v.errorUntil = time.Now().Add(5 * time.Second)
				ebiten.SetWindowTitle(videoWindowTitle + " - " + msg)
			}
		default:
			goto done
		}
	}
done:
	if v.lastError != "" && time.Now().After(v.errorUntil) {
		v.lastError = ""
		v.errorUntil = time.Time{}
		ebiten.SetWindowTitle(videoWindowTitle)
	}
}

func (v *videoPreview) Update() error {
	v.handleErrors()
	if err := v.handleInput(); err != nil {
		return err
	}
	width, height, pixels, ok := v.assembler.takeFrame()
	if ok {
		if width != v.width || height != v.height {
			v.width = width
			v.height = height
			v.minW = width
			v.minH = height
			v.nominalW = width * v.zoom
			v.nominalH = height * v.zoom
			if v.nominalW < 1 {
				v.nominalW = width
			}
			if v.nominalH < 1 {
				v.nominalH = height
			}
			ebiten.SetWindowSize(v.nominalW, v.nominalH)
			v.image = ebiten.NewImage(width, height)
			v.rgba = make([]byte, width*height*4)
		}
		if v.image != nil && len(v.rgba) == v.width*v.height*4 {
			for src, dst := 0, 0; src+2 < len(pixels) && dst+3 < len(v.rgba); src, dst = src+3, dst+4 {
				v.rgba[dst] = pixels[src]
				v.rgba[dst+1] = pixels[src+1]
				v.rgba[dst+2] = pixels[src+2]
				v.rgba[dst+3] = 0xFF
			}
			v.image.ReplacePixels(v.rgba)
		}
	}
	if v.minW > 0 && v.minH > 0 && (v.outsideW < v.minW || v.outsideH < v.minH) {
		if time.Since(v.lastResize) > 500*time.Millisecond {
			ebiten.SetWindowSize(v.nominalW, v.nominalH)
			v.lastResize = time.Now()
		}
	}
	return nil
}

func (v *videoPreview) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
	if v.image == nil || v.width < 1 || v.height < 1 {
		return
	}
	sw, sh := screen.Size()
	if sw < 1 || sh < 1 {
		return
	}
	scaleX := float64(sw) / float64(v.width)
	scaleY := float64(sh) / float64(v.height)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	newW := int(float64(v.width) * scale)
	newH := int(float64(v.height) * scale)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64((sw-newW)/2), float64((sh-newH)/2))
	screen.DrawImage(v.image, op)
}

func (v *videoPreview) Layout(outsideWidth, outsideHeight int) (int, int) {
	v.outsideW = outsideWidth
	v.outsideH = outsideHeight
	if outsideWidth < 1 {
		outsideWidth = v.width
	}
	if outsideHeight < 1 {
		outsideHeight = v.height
	}
	return outsideWidth, outsideHeight
}

func RunVideoPreview(host string, port int, refreshMS int, zoom int, socketPath string) error {
	if zoom < 1 {
		zoom = 1
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	assembler := &videoFrameAssembler{}
	stop := make(chan struct{})
	var inputCh chan inputEvent
	var errCh chan string
	go func() {
		buf := make([]byte, videoReadBufferBytes)
		for {
			_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			n, _, err := conn.ReadFrom(buf)
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					select {
					case <-stop:
						return
					default:
						continue
					}
				}
				return
			}
			assembler.process(buf[:n])
		}
	}()
	if socketPath != "" {
		inputCh = make(chan inputEvent, 64)
		errCh = make(chan string, 4)
		go func() {
			rpc := NewRpcClient(NewSocketTransport(socketPath))
			defer rpc.Close()
			for event := range inputCh {
				if err := rpc.InputKey(
					context.Background(),
					event.action,
					event.keyspace,
					event.mods,
					event.consol,
					event.keycode,
				); err != nil {
					select {
					case errCh <- err.Error():
					default:
					}
				}
			}
		}()
	}
	defer func() {
		close(stop)
		if inputCh != nil {
			close(inputCh)
		}
	}()

	tps := 60
	if refreshMS > 0 {
		tps = 1000 / refreshMS
		if tps < 1 {
			tps = 1
		}
	}
	ebiten.SetTPS(tps)
	ebiten.SetWindowTitle(videoWindowTitle)
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowSize(videoDefaultWidth*zoom, videoDefaultHeight*zoom)

	game := newVideoPreview(assembler, zoom, inputCh, errCh)
	return ebiten.RunGame(game)
}
