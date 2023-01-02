package vxproto

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// IModuleSocket is main interface for Module Socket integration
type IModuleSocket interface {
	GetName() string
	GetGroupID() string
	GetIMCToken() string
	GetReceiver() chan *Packet
	RecvData(ctx context.Context, timeout int64) (string, *Data, error)
	RecvFile(ctx context.Context, timeout int64) (string, *File, error)
	RecvText(ctx context.Context, timeout int64) (string, *Text, error)
	RecvMsg(ctx context.Context, timeout int64) (string, *Msg, error)
	RecvAction(ctx context.Context, timeout int64) (string, *Action, error)
	RecvDataFrom(ctx context.Context, src string, timeout int64) (*Data, error)
	RecvFileFrom(ctx context.Context, src string, timeout int64) (*File, error)
	RecvTextFrom(ctx context.Context, src string, timeout int64) (*Text, error)
	RecvMsgFrom(ctx context.Context, src string, timeout int64) (*Msg, error)
	RecvActionFrom(ctx context.Context, src string, timeout int64) (*Action, error)
	SendDataTo(ctx context.Context, dst string, data *Data) error
	SendFileTo(ctx context.Context, dst string, file *File) error
	SendTextTo(ctx context.Context, dst string, text *Text) error
	SendMsgTo(ctx context.Context, dst string, msg *Msg) error
	SendActionTo(ctx context.Context, dst string, act *Action) error
	Close(ctx context.Context)
	IRouter
	IIMC
}

var (
	ErrProtoIOUnset     = fmt.Errorf("the ProtoIO is nil")
	ErrDstUnreachable   = fmt.Errorf("destination is unreachable")
	ErrTopicUnreachable = fmt.Errorf("topic is unreachable")
	ErrDstMalformed     = fmt.Errorf("destination is not a module socket")
)

// moduleSocket is struct that used for receive data from other side to ms
type moduleSocket struct {
	name     string
	groupID  string
	imcToken string
	router   *recvRouter
	closer   func(ctx context.Context)
	IDefaultReceiver
	IProtoStats
	IProtoIO
	IRouter
	IIMC
}

// Close is function which stop all blockers and valid close this socket
func (ms *moduleSocket) Close(ctx context.Context) {
	if ms.router != nil {
		ms.router.close()
	}
	if ms.closer != nil {
		ms.closer(ctx)
	}
}

// GetName is function which using for access to private property name
func (ms *moduleSocket) GetName() string {
	return ms.name
}

// GetGroupID is function which using for access to private property groupID
func (ms *moduleSocket) GetGroupID() string {
	return ms.groupID
}

// GetIMCToken is function which using for access to private property imcToken
func (ms *moduleSocket) GetIMCToken() string {
	return ms.imcToken
}

// GetChannels is function which exported channels for receive packets and control
func (ms *moduleSocket) GetReceiver() chan *Packet {
	if ms.router == nil {
		return nil
	}
	return ms.router.receiver
}

// RecvData is function for receive Data packet with timeout (ms)
// If timeout equals -1 value so there will permanently blocked this function
func (ms *moduleSocket) RecvData(ctx context.Context, timeout int64) (string, *Data, error) {
	packet, err := ms.router.recvPacket(ctx, "", PTData, timeout)
	if err != nil {
		return "", nil, err
	}
	packet.SetAck()
	return packet.Src, packet.GetData(), nil
}

// RecvFile is function for receive File packet with timeout (ms)
// If timeout equals -1 value so there will permanently blocked this function
func (ms *moduleSocket) RecvFile(ctx context.Context, timeout int64) (string, *File, error) {
	packet, err := ms.router.recvPacket(ctx, "", PTFile, timeout)
	if err != nil {
		return "", nil, err
	}
	packet.SetAck()
	return packet.Src, packet.GetFile(), nil
}

// RecvText is function for receive Text packet with timeout (ms)
// If timeout equals -1 value so there will permanently blocked this function
func (ms *moduleSocket) RecvText(ctx context.Context, timeout int64) (string, *Text, error) {
	packet, err := ms.router.recvPacket(ctx, "", PTText, timeout)
	if err != nil {
		return "", nil, err
	}
	packet.SetAck()
	return packet.Src, packet.GetText(), nil
}

// RecvMsg is function for receive Msg packet with timeout (ms)
// If timeout equals -1 value so there will permanently blocked this function
func (ms *moduleSocket) RecvMsg(ctx context.Context, timeout int64) (string, *Msg, error) {
	packet, err := ms.router.recvPacket(ctx, "", PTMsg, timeout)
	if err != nil {
		return "", nil, err
	}
	packet.SetAck()
	return packet.Src, packet.GetMsg(), nil
}

// RecvAction is function for receive Action packet with timeout (ms)
// If timeout equals -1 value so there will permanently blocked this function
func (ms *moduleSocket) RecvAction(ctx context.Context, timeout int64) (string, *Action, error) {
	packet, err := ms.router.recvPacket(ctx, "", PTAction, timeout)
	if err != nil {
		return "", nil, err
	}
	packet.SetAck()
	return packet.Src, packet.GetAction(), nil
}

// RecvDataFrom is function for receive Data packet with timeout (ms)
// If src equals empty string so there will use receive from all agents and tokens
// If timeout equals -1 value so there will permanently blocked this function
func (ms *moduleSocket) RecvDataFrom(ctx context.Context, src string, timeout int64) (*Data, error) {
	packet, err := ms.router.recvPacket(ctx, src, PTData, timeout)
	if err != nil {
		return nil, err
	}
	packet.SetAck()
	return packet.GetData(), nil
}

// RecvFileFrom is function for receive File packet with timeout (ms)
// If src equals empty string so there will use receive from all agents and tokens
// If timeout equals -1 value so there will permanently blocked this function
func (ms *moduleSocket) RecvFileFrom(ctx context.Context, src string, timeout int64) (*File, error) {
	packet, err := ms.router.recvPacket(ctx, src, PTFile, timeout)
	if err != nil {
		return nil, err
	}
	packet.SetAck()
	return packet.GetFile(), nil
}

// RecvTextFrom is function for receive Text packet with timeout (ms)
// If src equals empty string so there will use receive from all agents and tokens
// If timeout equals -1 value so there will permanently blocked this function
func (ms *moduleSocket) RecvTextFrom(ctx context.Context, src string, timeout int64) (*Text, error) {
	packet, err := ms.router.recvPacket(ctx, src, PTText, timeout)
	if err != nil {
		return nil, err
	}
	packet.SetAck()
	return packet.GetText(), nil
}

// RecvMsgFrom is function for receive Msg packet with timeout (ms)
// If src equals empty string so there will use receive from all agents and tokens
// If timeout equals -1 value so there will permanently blocked this function
func (ms *moduleSocket) RecvMsgFrom(ctx context.Context, src string, timeout int64) (*Msg, error) {
	packet, err := ms.router.recvPacket(ctx, src, PTMsg, timeout)
	if err != nil {
		return nil, err
	}
	packet.SetAck()
	return packet.GetMsg(), nil
}

// RecvActionFrom is function for receive Action packet with timeout (ms)
// If src equals empty string so there will use receive from all agents and tokens
// If timeout equals -1 value so there will permanently blocked this function
func (ms *moduleSocket) RecvActionFrom(ctx context.Context, src string, timeout int64) (*Action, error) {
	packet, err := ms.router.recvPacket(ctx, src, PTAction, timeout)
	if err != nil {
		return nil, err
	}
	packet.SetAck()
	return packet.GetAction(), nil
}

func (ms *moduleSocket) sendFileStream(ctx context.Context, dst string, file *File) error {
	uniqRaw := make([]byte, 32)
	if _, err := rand.Read(uniqRaw); err != nil {
		return fmt.Errorf("failed to get a random unique value: %w", err)
	}
	uniqHash := md5.Sum(uniqRaw)
	uniq := hex.EncodeToString(uniqHash[:]) + ":"
	sendFile := func(dataLen int, read func(left, right int) ([]byte, error)) error {
		nPackets := dataLen / MaxFilePacketChunkSize
		if nPackets*MaxFilePacketChunkSize < dataLen || dataLen == 0 {
			nPackets += 1
		}
		lastN := strconv.Itoa(nPackets)
		for ix := 0; ix < nPackets; ix++ {
			left := ix * MaxFilePacketChunkSize
			right := (ix + 1) * MaxFilePacketChunkSize
			if right > dataLen {
				right = dataLen
			}
			curN := strconv.Itoa(ix + 1)
			data, err := read(left, right)
			if err != nil {
				return fmt.Errorf("the read callback has returned an error: %w", err)
			}
			part := &File{
				Data: data,
				Name: file.Name,
				Uniq: uniq + curN + ":" + lastN,
			}
			if err := ms.sendPacket(ctx, dst, PTFile, part); err != nil {
				return fmt.Errorf("failed to send a packet: %w", err)
			}
		}
		return nil
	}
	dataLen := len(file.Data)
	checkIndices := func(left int, right int, dataLen int) error {
		if left < 0 {
			return fmt.Errorf("invalid left index %d: cannot be negative", left)
		}
		if left > right {
			return fmt.Errorf("invalid left (%d) and right (%d) indices: the left index cannot be greater than the right index", left, right)
		}
		if right > dataLen {
			return fmt.Errorf("invalid right index (%d) for the given data length (%d): the right index cannot be greater than the data length", right, dataLen)
		}
		return nil
	}
	if dataLen != 0 || len(file.Path) == 0 {
		// Send raw data
		read := func(left, right int) ([]byte, error) {
			if err := checkIndices(left, right, dataLen); err != nil {
				return nil, err
			}
			return file.Data[left:right], nil
		}
		return sendFile(dataLen, read)
	} else {
		// Send data from file path
		fh, err := os.Open(file.Path)
		if err != nil {
			return fmt.Errorf("failed to open the file '%s': %w", file.Path, err)
		}
		defer fh.Close()
		fs, err := fh.Stat()
		if err != nil {
			return fmt.Errorf("failed to get the file '%s' stats: %w", file.Path, err)
		}
		dataLen = int(fs.Size())
		read := func(left, right int) ([]byte, error) {
			if err := checkIndices(left, right, dataLen); err != nil {
				return nil, err
			}
			buf := make([]byte, right-left)
			if nb, err := fh.ReadAt(buf, int64(left)); err != nil {
				if err == io.EOF {
					return make([]byte, 0), nil
				}
				return nil, fmt.Errorf("failed to read the file part: %w", err)
			} else if nb != right-left {
				return nil, fmt.Errorf("invalid bytes length to read data")
			}
			return buf, nil
		}
		return sendFile(dataLen, read)
	}
}

func (ms *moduleSocket) updateMeta(metaFile, chunk, total string) (bool, error) {
	var chunks []string
	metaFileFlags := os.O_RDWR | os.O_CREATE
	if metaFileHandle, err := os.OpenFile(metaFile, metaFileFlags, 0600); err != nil {
		return false, err
	} else {
		errReturn := func(err error) (bool, error) {
			metaFileHandle.Close()
			return false, err
		}

		metaFileData, err := ioutil.ReadAll(metaFileHandle)
		if err != nil {
			return errReturn(err)
		}
		if len(metaFileData) != 0 {
			if err = json.Unmarshal(metaFileData, &chunks); err != nil {
				return errReturn(err)
			}
		}

		chunks = append(chunks, chunk)
		if metaFileData, err = json.Marshal(&chunks); err != nil {
			return errReturn(err)
		}
		if n, err := metaFileHandle.WriteAt(metaFileData, 0); err != nil {
			return errReturn(err)
		} else if n != len(metaFileData) {
			return errReturn(fmt.Errorf("failed to update the whole meta file"))
		}

		if err = metaFileHandle.Close(); err != nil {
			return false, err
		}
	}

	totalPackets, err := strconv.Atoi(total)
	if err != nil {
		return false, err
	}
	if len(chunks) == totalPackets {
		if err = os.Remove(metaFile); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (ms *moduleSocket) parseFileStream(ctx context.Context, packet *Packet) (bool, error) {
	file := packet.GetFile()
	if file == nil {
		return false, fmt.Errorf("failed to cast the packet to the file")
	}

	if file.Uniq == "" {
		// Send direct messages as a file type (only internal communication)
		file.Path = ""
		return true, nil
	}

	uniqArr := strings.Split(file.Uniq, ":")
	if file.Data == nil && file.Path != "" && file.Uniq != "" && len(uniqArr) == 1 {
		// Case for multiple sending one file packet to other receivers via topic
		return true, nil
	} else if len(uniqArr) != 3 {
		return false, fmt.Errorf("failed to parse the packet unique identifier")
	}

	tempDir := filepath.Join(os.TempDir(), "vx-store")
	os.Mkdir(tempDir, 0700)

	tempFile := filepath.Join(tempDir, uniqArr[0])
	tempFileFlags := os.O_WRONLY | os.O_CREATE
	if fileHandle, err := os.OpenFile(tempFile, tempFileFlags, 0600); err != nil {
		return false, err
	} else {
		defer fileHandle.Close()

		curPacket, err := strconv.Atoi(uniqArr[1])
		if err != nil {
			return false, err
		}
		offset := int64((curPacket - 1) * MaxFilePacketChunkSize)

		if n, err := fileHandle.WriteAt(file.Data, offset); err != nil {
			return false, err
		} else if n != len(file.Data) {
			return false, fmt.Errorf("failed to write the whole packet data")
		}

		file.Data = nil
		file.Path = tempFile
		file.Uniq = uniqArr[0]
	}

	metaFile := filepath.Join(tempDir, uniqArr[0]+".meta")
	return ms.updateMeta(metaFile, uniqArr[1], uniqArr[2])
}

// recvPacket is function for receiving from agent and serving packet to target module
// Result is the success of packet processing otherwise will raise error
func (ms *moduleSocket) recvPacket(ctx context.Context, packet *Packet) error {
	// storing file into FS before notify receiver about it
	if packet.PType == PTFile {
		if ok, err := ms.parseFileStream(ctx, packet); err != nil {
			return err
		} else if !ok {
			// Skip forwarding packet if it is not finish packet
			return nil
		}
	}

	if ms.IDefaultReceiver != nil {
		switch packet.PType {
		case PTFile, PTText, PTMsg, PTAction:
			if err := ms.DefaultRecvPacket(ctx, packet); err != nil {
				return err
			}
		}
	}
	return ms.router.routePacket(ctx, packet)
}

// sendPacket use in ms API to send data to agent
func (ms *moduleSocket) sendPacket(ctx context.Context, dst string, pType PacketType, payload interface{}) error {
	packet := &Packet{
		ctx:     ctx,
		Module:  ms.GetName(),
		Dst:     dst,
		TS:      time.Now().Unix(),
		PType:   pType,
		Payload: payload,
	}
	if ms.IProtoIO == nil {
		return ErrProtoIOUnset
	}
	sendViaIMC := func(token string) error {
		idstms := ms.GetIMCModule(token)
		if idstms == nil {
			return ErrDstUnreachable
		}
		dstms, ok := idstms.(*moduleSocket)
		if !ok {
			return ErrDstMalformed
		}
		packet.Module = dstms.GetName()
		packet.Src = ms.GetIMCToken()
		packet.ackChan = make(chan struct{}, 1)
		packet.ctx = ctx
		return dstms.recvPacket(ctx, packet)
	}
	if ms.HasIMCTopicFormat(dst) {
		iti := ms.GetIMCTopic(dst)
		if iti == nil {
			return ErrTopicUnreachable
		}
		var lastErr error
		for _, token := range iti.GetSubscriptions() {
			if err := sendViaIMC(token); err != nil {
				lastErr = err
			}
		}
		return lastErr
	}
	if ms.HasIMCTokenFormat(dst) {
		return sendViaIMC(dst)
	}
	return ms.IProtoIO.sendPacket(ctx, packet)
}

// SendDataTo use in ms API to send data to agent
func (ms *moduleSocket) SendDataTo(ctx context.Context, dst string, data *Data) error {
	return ms.sendPacket(ctx, dst, PTData, data)
}

// SendFileTo use in ms API to send file to agent
func (ms *moduleSocket) SendFileTo(ctx context.Context, dst string, file *File) error {
	return ms.sendFileStream(ctx, dst, file)
}

// SendTextTo use in ms API to send text to agent
func (ms *moduleSocket) SendTextTo(ctx context.Context, dst string, text *Text) error {
	return ms.sendPacket(ctx, dst, PTText, text)
}

// SendMsgTo use in ms API to send message to agent
func (ms *moduleSocket) SendMsgTo(ctx context.Context, dst string, msg *Msg) error {
	return ms.sendPacket(ctx, dst, PTMsg, msg)
}

// SendActionTo use in ms API to send action to agent
func (ms *moduleSocket) SendActionTo(ctx context.Context, dst string, act *Action) error {
	return ms.sendPacket(ctx, dst, PTAction, act)
}
