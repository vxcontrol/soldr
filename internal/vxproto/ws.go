//nolint:staticcheck
package vxproto

//TODO: io/ioutil is deprecated, replace to fs.FS and delete "nolint:staticcheck"
import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var PacketMarkerV1 = []byte{0xf0, 0xa0, 0x60, 0x30}

// IWSConnection is extended gorilla websocket interface
// to synchronized Read and Write functions.
// It's goroutine safe implementation only for Read and Write.
type IWSConnection interface {
	Close(ctx context.Context) error
	CloseHandler() func(code int, text string) error
	EnableWriteCompression(enable bool)
	LocalAddr() net.Addr
	NextReader() (messageType int, r io.Reader, err error)
	NextWriter(messageType int) (io.WriteCloser, error)
	PingHandler() func(appData string) error
	PongHandler() func(appData string) error
	Read(ctx context.Context) (data []byte, err error)
	ReadMessage() (messageType int, p []byte, err error)
	RemoteAddr() net.Addr
	SetCloseHandler(h func(code int, text string) error)
	SetCompressionLevel(level int) error
	SetPingHandler(h func(appData string) error)
	SetPongHandler(h func(appData string) error)
	SetReadDeadline(t time.Time) error
	SetReadLimit(limit int64)
	SetWriteDeadline(t time.Time) error
	Subprotocol() string
	UnderlyingConn() net.Conn
	Write(ctx context.Context, data []byte) error
	WriteControl(messageType int, data []byte, deadline time.Time) error
	WriteMessage(messageType int, data []byte) error
	WritePreparedMessage(pm *websocket.PreparedMessage) error
}

// wsConnection is implementatation of websocket connection for IConnection
type wsConnection struct {
	buffer    []byte
	closed    bool
	closing   bool
	original  bool
	isWDSet   bool
	isWDSetMX *sync.Mutex
	closeMX   *sync.Mutex
	recvMX    *sync.Mutex
	sendMX    *sync.Mutex
	*websocket.Conn
}

// NewWSConnection is constructor for new vproto adaptive WS connection
func NewWSConnection(conn *websocket.Conn, original bool) IWSConnection {
	return &wsConnection{
		original:  original,
		closeMX:   &sync.Mutex{},
		recvMX:    &sync.Mutex{},
		sendMX:    &sync.Mutex{},
		isWDSet:   false,
		isWDSetMX: &sync.Mutex{},
		Conn:      conn,
	}
}

// Read is synchronous function that provides interaction with the websocket API
func (ws *wsConnection) Read(ctx context.Context) (data []byte, err error) {
	var (
		msgType int
		reader  io.Reader
	)

	if ws.checkClosed() {
		err = fmt.Errorf("connection is not initialized: %w", websocket.ErrCloseSent)
		return
	}

	defer func() {
		if err != nil {
			ws.Close(ctx)
		}
	}()

	ws.recvMX.Lock()
	defer ws.recvMX.Unlock()

	if ws.buffer == nil {
		if msgType, reader, err = ws.NextReader(); err != nil {
			err = fmt.Errorf("failed to get connection reader: %w", err)
			return
		}
		if msgType != websocket.BinaryMessage {
			err = fmt.Errorf("unexpected message type: %d", msgType)
			return
		}
		if reader == nil {
			err = fmt.Errorf("failed to initialize connection reader")
			return
		}

		if ws.buffer, err = ioutil.ReadAll(reader); err != nil {
			err = fmt.Errorf("failed to get data from connection reader: %w", err)
			return
		}
		if ws.buffer == nil {
			err = fmt.Errorf("failed to initialize reader buffer")
			return
		}
	}

	if ws.original {
		data, ws.buffer = ws.buffer, nil
		return
	}

	if len(ws.buffer) < 8 {
		ws.original = true
		data, ws.buffer = ws.buffer, nil
		return
	}

	// TODO: change it after than all agents will moved to new proto
	// here need to raise error
	if !bytes.HasPrefix(ws.buffer, []byte(PacketMarkerV1)) {
		ws.original = true
		data, ws.buffer = ws.buffer, nil
		return
	}
	packetLen := int(binary.LittleEndian.Uint32(ws.buffer[4:8]))
	if len(ws.buffer)-8 < packetLen {
		ws.original = true
		data, ws.buffer = ws.buffer, nil
		return
	} else if len(ws.buffer)-8 == packetLen {
		data, ws.buffer = ws.buffer[8:], nil
		return
	} else {
		data, ws.buffer = ws.buffer[8:packetLen+8], ws.buffer[packetLen+8:]
		return
	}
}

func (ws *wsConnection) SetWriteDeadline(t time.Time) error {
	ws.isWDSetMX.Lock()
	defer ws.isWDSetMX.Unlock()

	if err := ws.Conn.SetWriteDeadline(t); err != nil {
		return err
	}
	ws.isWDSet = true
	return nil
}

func (ws *wsConnection) resetWriteDeadline() error {
	ws.isWDSetMX.Lock()
	defer ws.isWDSetMX.Unlock()

	if ws.isWDSet {
		return nil
	}
	const defaultWSWriteTimeout = time.Second * 60
	if err := ws.Conn.SetWriteDeadline(time.Now().Add(defaultWSWriteTimeout)); err != nil {
		return fmt.Errorf("failed to set the write deadline: %w", err)
	}
	return nil
}

// Write is synchronous function that provides interaction with the websocket API
func (ws *wsConnection) Write(ctx context.Context, data []byte) (err error) {
	if ws.checkClosed() {
		err = fmt.Errorf("connection is not initialized: %w", websocket.ErrCloseSent)
		return
	}

	defer func() {
		if err != nil {
			ws.Close(ctx)
		}
	}()

	ws.sendMX.Lock()
	defer ws.sendMX.Unlock()

	if err := ws.resetWriteDeadline(); err != nil {
		return err
	}

	var writer io.WriteCloser
	writer, err = ws.NextWriter(websocket.BinaryMessage)
	if err != nil {
		err = fmt.Errorf("failed to get connection writer: %w", err)
		return
	}
	if writer == nil {
		err = fmt.Errorf("failed to initialize connection writer")
		return
	}
	if !ws.original {
		prefixBuffer := new(bytes.Buffer)
		e := binary.Write(prefixBuffer, binary.LittleEndian, PacketMarkerV1)
		if e != nil {
			logrus.Errorf("binary.Write failed: %s", e)
		}
		e = binary.Write(prefixBuffer, binary.LittleEndian, uint32(len(data)))
		if e != nil {
			logrus.Errorf("binary.Write failed: %s", e)
		}
		prefix := prefixBuffer.Bytes()
		if n, err := writer.Write(prefix); err != nil || n != len(prefix) {
			return fmt.Errorf(
				"failed to write the message prefix (symbols actually written: %d, expected: %d): %w",
				n, len(prefix), err,
			)
		}
	}
	if n, err := writer.Write(data); err != nil || n != len(data) {
		return fmt.Errorf(
			"failed to write the message payload (symbols actually written: %d, expected: %d): %w",
			n, len(data), err,
		)
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("failed to close packet writer: %w", err)
	}
	return nil
}

// ReadMessage is a helper method for getting a reader using NextReader and
// reading from that reader to a buffer.
func (ws *wsConnection) ReadMessage() (messageType int, p []byte, err error) {
	if ws.checkClosed() {
		return messageType, nil, fmt.Errorf("connection is not initialized: %w", websocket.ErrCloseSent)
	}

	ws.recvMX.Lock()
	defer ws.recvMX.Unlock()

	return ws.Conn.ReadMessage()
}

// WriteMessage is a helper method for getting a writer using NextWriter,
// writing the message and closing the writer.
func (ws *wsConnection) WriteMessage(messageType int, data []byte) error {
	if ws.checkClosed() {
		return fmt.Errorf("connection is not initialized: %w", websocket.ErrCloseSent)
	}

	ws.sendMX.Lock()
	defer ws.sendMX.Unlock()

	if err := ws.resetWriteDeadline(); err != nil {
		return err
	}

	return ws.Conn.WriteMessage(messageType, data)
}

// WritePreparedMessage writes prepared message into connection.
func (ws *wsConnection) WritePreparedMessage(pm *websocket.PreparedMessage) error {
	if ws.checkClosed() {
		return fmt.Errorf("connection is not initialized: %w", websocket.ErrCloseSent)
	}

	ws.sendMX.Lock()
	defer ws.sendMX.Unlock()

	if err := ws.resetWriteDeadline(); err != nil {
		return err
	}

	return ws.Conn.WritePreparedMessage(pm)
}

// Close is function to close websocket correctly
func (ws *wsConnection) Close(ctx context.Context) error {
	if !ws.startClosing() {
		return nil
	}

	defer func() {
		ws.setClosed()
		// the receiving reader may lose some message when the sender closes the channel
		_ = ws.Conn.Close()
	}()

	if err := ws.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	); err != nil {
		if strings.HasPrefix(err.Error(), "websocket: close ") {
			// to avoid issue with hanging TLS/TCP connection
			_ = ws.Conn.Close()
			return nil
		}
		err = fmt.Errorf("failed to write a request Close control message: %w", err)
		if closeErr := ws.Conn.Close(); closeErr != nil {
			err = fmt.Errorf(
				"failed to properly close the websocket connection (%v) after another error has occurred: %w",
				closeErr,
				err,
			)
		}
		return err
	}

	return nil
}

func (ws *wsConnection) startClosing() bool {
	ws.closeMX.Lock()
	defer ws.closeMX.Unlock()

	if ws.closing {
		return false
	}
	ws.closing = true
	return true
}

func (ws *wsConnection) checkClosed() bool {
	ws.closeMX.Lock()
	defer ws.closeMX.Unlock()

	return ws.closed
}

func (ws *wsConnection) setClosed() {
	ws.closeMX.Lock()
	defer ws.closeMX.Unlock()

	ws.closed = true
}
