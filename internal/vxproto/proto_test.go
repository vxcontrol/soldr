package vxproto

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"soldr/internal/protoagent"
	"soldr/internal/system"
	"soldr/internal/utils"
	"soldr/internal/vxproto/tunnel"
	tunnelRC4 "soldr/internal/vxproto/tunnel/rc4"
)

var errForceClosed = fmt.Errorf("connection was force closed")

type FakeSocket struct {
	isConnected bool
	RecvChan    chan []byte
	SendChan    chan []byte
	Ctx         context.Context
	CancelCtx   context.CancelFunc
}

func (fc *FakeSocket) Read(ctx context.Context) (data []byte, err error) {
	if fc.isConnected {
		select {
		case msg := <-fc.RecvChan:
			return msg, nil
		case <-fc.Ctx.Done():
			return nil, errForceClosed
		}
	} else {
		return nil, fmt.Errorf("connection has already closed: %w", websocket.ErrCloseSent)
	}
}

func (fc *FakeSocket) Write(ctx context.Context, data []byte) error {
	if fc.isConnected {
		select {
		case fc.SendChan <- data:
			return nil
		case <-fc.Ctx.Done():
			return errForceClosed
		}
	} else {
		return fmt.Errorf("connection has already closed: %w", websocket.ErrCloseSent)
	}
}

func (fc *FakeSocket) Close(ctx context.Context) error {
	if fc.isConnected {
		fc.isConnected = false
		fc.CancelCtx()
	} else {
		return fmt.Errorf("connection has already closed: %w", websocket.ErrCloseSent)
	}
	return nil
}

type FakeMainModule struct{}

func (mModule *FakeMainModule) DefaultRecvPacket(ctx context.Context, packet *Packet) error {
	return nil
}

func (mModule *FakeMainModule) HasAgentInfoValid(ctx context.Context, iasocket IAgentSocket) error {
	return nil
}

func (mModule *FakeMainModule) GetVersion() string {
	return "develop"
}

const (
	agentID     = "12345678901234567890123456789012"
	groupID     = "89012345678901234567890123456789"
	agentToken  = "1234567890123456789012345678901234567890"
	serverToken = "0123456789012345678901234567890123456789"
)

var packEncrypter, _ = tunnel.NewPackEncrypter(&tunnel.Config{
	RC4: &tunnelRC4.Config{
		Key: []byte{42},
	},
})

func makeAgentSocket(vxp *vxProto) *agentSocket {
	seconds := time.Now().Unix()
	user := &protoagent.Information_User{
		Name:   utils.GetRef("root"),
		Groups: []string{"root"},
	}
	return &agentSocket{
		id:  agentID,
		gid: groupID,
		ip:  "192.168.1.1",
		src: serverToken,
		at:  VXAgent,
		auth: &AuthenticationData{
			req: &protoagent.AuthenticationRequest{
				Timestamp: &seconds,
				Atoken:    utils.GetRef(""),
				Aversion:  utils.GetRef(vxp.GetVersion()),
			},
			resp: &protoagent.AuthenticationResponse{
				Atoken:   utils.GetRef(agentToken),
				Stoken:   utils.GetRef(serverToken),
				Sversion: utils.GetRef(vxp.GetVersion()),
				Status:   utils.GetRef("authorized"),
			},
		},
		info: &protoagent.Information{
			Os: &protoagent.Information_OS{
				Type: utils.GetRef("linux"),
				Name: utils.GetRef("Ubuntu 16.04"),
				Arch: utils.GetRef("amd64"),
			},
			Net: &protoagent.Information_Net{
				Hostname: utils.GetRef("test_pc"),
				Ips:      []string{"127.0.0.1/8"},
			},
			Users: []*protoagent.Information_User{
				user,
			},
		},
		packEncrypter:    packEncrypter,
		IConnection:      &FakeSocket{},
		IVaildator:       vxp,
		IMMInformator:    vxp,
		IProtoStats:      vxp,
		IProtoIO:         vxp,
		connectionPolicy: newAllowPacketChecker(),
	}
}

func makePacket(pType PacketType, payload interface{}) *Packet {
	seconds := time.Now().Unix()
	packet := &Packet{
		ctx:     context.Background(),
		Module:  "test",
		Src:     agentToken,
		Dst:     serverToken,
		TS:      seconds,
		PType:   pType,
		Payload: payload,
	}
	return packet
}

func makePacketData(data []byte) *Packet {
	return makePacket(PTData, &Data{Data: data})
}

func makePacketText(data []byte, name string) *Packet {
	return makePacket(PTText, &Text{Data: data, Name: name})
}

func makePacketFile(data []byte, path, name string) *Packet {
	return makePacket(PTFile, &File{Data: data, Name: name, Path: path})
}

func makePacketMsg(data []byte, mType MsgType) *Packet {
	return makePacket(PTMsg, &Msg{Data: data, MType: mType})
}

func makePacketAction(data []byte, name string) *Packet {
	return makePacket(PTAction, &Action{Data: data, Name: name})
}

func checkFileData(path string, data []byte) bool {
	fdata, err := ioutil.ReadFile(path)
	if err != nil {
		return false
	}
	return bytes.Equal(fdata, data)
}

func parsePacket(data []byte, pType PacketType) (*Packet, error) {
	packet, err := (&Packet{}).fromBytesPB(data)
	if err != nil {
		return nil, err
	}

	if packet.Module != "test" {
		return nil, fmt.Errorf("invalid module name")
	}

	if packet.Dst != agentToken {
		return nil, fmt.Errorf("invalid destination")
	}

	if packet.PType != pType {
		return nil, fmt.Errorf("invalid packet type")
	}

	return packet, nil
}

func parsePacketData(data []byte) ([]byte, error) {
	packet, err := parsePacket(data, PTData)
	if err != nil {
		return nil, err
	}

	return packet.GetData().Data, nil
}

func parsePacketText(data []byte) ([]byte, string, error) {
	packet, err := parsePacket(data, PTText)
	if err != nil {
		return nil, "", err
	}

	return packet.GetText().Data, packet.GetText().Name, nil
}

func parsePacketFile(data []byte) ([]byte, string, string, error) {
	packet, err := parsePacket(data, PTFile)
	if err != nil {
		return nil, "", "", err
	}

	return packet.GetFile().Data, packet.GetFile().Path, packet.GetFile().Name, nil
}

func parsePacketMsg(data []byte) ([]byte, MsgType, error) {
	packet, err := parsePacket(data, PTMsg)
	if err != nil {
		return nil, 0, err
	}

	return packet.GetMsg().Data, packet.GetMsg().MType, nil
}

func parsePacketAction(data []byte) ([]byte, string, error) {
	packet, err := parsePacket(data, PTAction)
	if err != nil {
		return nil, "", err
	}

	return packet.GetAction().Data, packet.GetAction().Name, nil
}

func randString(nchars int) string {
	rbytes := make([]byte, nchars)
	if _, err := rand.Read(rbytes); err != nil {
		return ""
	}

	return hex.EncodeToString(rbytes)
}

func TestNewProto(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	if err := proto.Close(ctx); err != nil {
		t.Fatal(err.Error())
	}
}

func TestTokenValidation(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}

	agentID := system.MakeAgentID()
	agentSocket := makeAgentSocket(proto)
	for _, agentType := range []AgentType{VXAgent, VXServer, Browser, External} {
		token, err := agentSocket.NewToken(agentID, agentType)
		if token == "" || err != nil {
			t.Fatal("Failed generate token")
		}
		if !agentSocket.HasTokenCRCValid(token) {
			t.Fatal("Failed validate CRC32 of token")
		}
		if !agentSocket.HasTokenValid(token, agentID, agentType) {
			t.Fatal("Failed full validate of token")
		}
	}

	if err := proto.Close(ctx); err != nil {
		t.Fatal(err.Error())
	}
}

func TestTokenGeneration(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}

	agentID := system.MakeAgentID()
	agentSocket := makeAgentSocket(proto)
	for _, agentType := range []AgentType{VXAgent, VXServer, Browser, External} {
		token1, err := agentSocket.NewToken(agentID, agentType)
		if token1 == "" || err != nil {
			t.Fatal("Failed generate first token")
		}
		token2, err := agentSocket.NewToken(agentID, agentType)
		if token2 == "" || err != nil {
			t.Fatal("Failed generate second token")
		}
		switch agentType {
		case VXAgent, VXServer:
			// must be identical
			if token1 != token2 {
				t.Fatal("Failed to match tokens with VXAgent type")
			}
		case Browser, External:
			// must be different
			if token1 == token2 {
				t.Fatal("Failed to match tokens with non VXAgent type")
			}
		}
	}

	if err := proto.Close(ctx); err != nil {
		t.Fatal(err.Error())
	}
}

func TestNewModule(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	if moduleSocket := proto.NewModule("test", groupID); moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			t.Fatal("Failed add Module Socket object")
		}
		if !proto.DelModule(moduleSocket) {
			t.Fatal("Failed delete Module Socket object")
		}
	} else {
		t.Fatal("Failed initialize Module Socket object")
	}
	if err := proto.Close(ctx); err != nil {
		t.Fatal(err.Error())
	}
}

func TestLinkAgentToModule(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket := proto.NewModule("test", groupID)
	if moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			t.Fatal("Failed add Module Socket object")
		}
	} else {
		t.Fatal("Failed initialize Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket) {
			t.Fatal("Failed delete Module Socket object")
		}
	}()

	var wg sync.WaitGroup
	agentSocket := makeAgentSocket(proto)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if !proto.addAgent(ctx, agentSocket) {
			t.Error("Failed add Agent Socket object")
		}
		defer func() {
			if !proto.delAgent(ctx, agentSocket) {
				t.Error("Failed delete Agent Socket object")
			}
		}()
	}()

	var packet *Packet
	packet = <-moduleSocket.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentConnected || packet.GetControlMsg().AgentInfo.ID != agentID {
		t.Fatal("Failed connect Agent to proto")
	}
	packet.SetAck()

	packet = <-moduleSocket.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentDisconnected || packet.GetControlMsg().AgentInfo.ID != agentID {
		t.Fatal("Failed disconnect Agent from proto")
	}
	packet.SetAck()

	wg.Wait()
}

func TestAPISendFromAgentToModule(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket := proto.NewModule("test", groupID)
	if moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			t.Fatal("Failed add Module Socket object")
		}
	} else {
		t.Fatal("Failed initialize Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket) {
			t.Fatal("Failed delete Module Socket object")
		}
	}()

	var wg sync.WaitGroup
	numPackets := 5
	testData := []byte("test data message")
	actName := "test_action_name"
	textName := "test_text_name"
	fileName := "test_file_name.tmp"
	msgType := MTDebug
	agentSocket := makeAgentSocket(proto)
	recvSyncChan := make(chan struct{})
	sendSyncChan := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		packetData, err := moduleSocket.RecvDataFrom(ctx, agentToken, -1)
		if err != nil {
			t.Error("Failed receive data packet")
		}
		if !bytes.Equal(packetData.Data, testData) {
			t.Error("Failed compare of received data")
		}
		recvSyncChan <- struct{}{}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		packetText, err := moduleSocket.RecvTextFrom(ctx, agentToken, -1)
		if err != nil {
			t.Error("Failed receive text packet")
		}
		if !bytes.Equal(packetText.Data, testData) || packetText.Name != textName {
			t.Error("Failed compare of received text")
		}
		recvSyncChan <- struct{}{}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		packetFile, err := moduleSocket.RecvFileFrom(ctx, agentToken, -1)
		if err != nil {
			t.Error("Failed receive file packet")
		}
		if !bytes.Equal(packetFile.Data, testData) || packetFile.Name != fileName || packetFile.Path != "" {
			t.Error("Failed compare of received file")
		}
		recvSyncChan <- struct{}{}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		packetMsg, err := moduleSocket.RecvMsgFrom(ctx, agentToken, -1)
		if err != nil {
			t.Error("Failed receive message packet")
		}
		if !bytes.Equal(packetMsg.Data, testData) || packetMsg.MType != msgType {
			t.Error("Failed compare of received message")
		}
		recvSyncChan <- struct{}{}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		packetAct, err := moduleSocket.RecvActionFrom(ctx, agentToken, -1)
		if err != nil {
			t.Error("Failed receive action packet")
		}
		if !bytes.Equal(packetAct.Data, testData) || packetAct.Name != actName {
			t.Error("Failed compare of received action")
		}
		recvSyncChan <- struct{}{}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if !proto.addAgent(ctx, agentSocket) {
			t.Error("Failed add Agent Socket object")
		}
		defer func() {
			if !proto.delAgent(ctx, agentSocket) {
				t.Error("Failed delete Agent Socket object")
			}
		}()

		addPacket := func(packet *Packet) {
			if err := proto.recvPacket(ctx, packet); err != nil {
				t.Error("Failed to receive packet:", err.Error())
			}
		}

		time.Sleep(100 * time.Millisecond)
		addPacket(makePacketData(testData))
		addPacket(makePacketText(testData, textName))
		addPacket(makePacketFile(testData, "", fileName))
		addPacket(makePacketMsg(testData, msgType))
		addPacket(makePacketAction(testData, actName))
		for i := 0; i < numPackets; i++ {
			<-recvSyncChan
		}
		<-sendSyncChan

		addPacket(makePacketData(testData))
		addPacket(makePacketText(testData, textName))
		addPacket(makePacketFile(testData, "", fileName))
		addPacket(makePacketMsg(testData, msgType))
		addPacket(makePacketAction(testData, actName))
		<-sendSyncChan
	}()

	var packet *Packet
	packet = <-moduleSocket.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentConnected || packet.GetControlMsg().AgentInfo.ID != agentID {
		t.Fatal("Failed connect Agent to proto")
	}
	packet.SetAck()
	sendSyncChan <- struct{}{}

	packet = <-moduleSocket.GetReceiver()
	if !bytes.Equal(packet.GetData().Data, testData) {
		t.Fatal("Failed compare of received data")
	}
	packet.SetAck()
	packet = <-moduleSocket.GetReceiver()
	if !bytes.Equal(packet.GetText().Data, testData) || packet.GetText().Name != textName {
		t.Fatal("Failed compare of received text")
	}
	packet.SetAck()
	packet = <-moduleSocket.GetReceiver()
	if !bytes.Equal(packet.GetFile().Data, testData) || packet.GetFile().Name != fileName || packet.GetFile().Path != "" {
		t.Fatal("Failed compare of received file")
	}
	packet.SetAck()
	packet = <-moduleSocket.GetReceiver()
	if !bytes.Equal(packet.GetMsg().Data, testData) || packet.GetMsg().MType != msgType {
		t.Fatal("Failed compare of received message")
	}
	packet.SetAck()
	packet = <-moduleSocket.GetReceiver()
	if !bytes.Equal(packet.GetAction().Data, testData) || packet.GetAction().Name != actName {
		t.Fatal("Failed compare of received action")
	}
	packet.SetAck()

	sendSyncChan <- struct{}{}

	packet = <-moduleSocket.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentDisconnected || packet.GetControlMsg().AgentInfo.ID != agentID {
		t.Fatal("Failed disconnect Agent from proto")
	}
	packet.SetAck()

	wg.Wait()
}

func TestDropAgent(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket := proto.NewModule("test", groupID)
	if moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			t.Fatal("Failed add Module Socket object")
		}
	} else {
		t.Fatal("Failed initialize Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket) {
			t.Fatal("Failed delete Module Socket object")
		}
	}()

	var wg sync.WaitGroup
	ctx, cancelCtx := context.WithCancel(ctx)
	fakeSocket := &FakeSocket{
		isConnected: true,
		RecvChan:    make(chan []byte),
		SendChan:    make(chan []byte),
		Ctx:         ctx,
		CancelCtx:   cancelCtx,
	}
	agentSocket := makeAgentSocket(proto)
	agentSocket.IConnection = fakeSocket

	wg.Add(1)
	go func() {
		defer wg.Done()
		if !proto.addAgent(ctx, agentSocket) {
			t.Error("Failed add Agent Socket object")
		}
		<-fakeSocket.Ctx.Done()
		if proto.delAgent(ctx, agentSocket) {
			t.Error("Failed delete removed Agent Socket object")
		}
	}()

	var packet *Packet
	packet = <-moduleSocket.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentConnected ||
		packet.GetControlMsg().AgentInfo.ID != agentID ||
		packet.GetControlMsg().AgentInfo.GID != groupID {
		t.Fatal("Failed connect Agent to module socket 1")
	}
	packet.SetAck()

	proto.DropAgent(ctx, agentID)

	packet = <-moduleSocket.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentDisconnected ||
		packet.GetControlMsg().AgentInfo.ID != agentID ||
		packet.GetControlMsg().AgentInfo.GID != groupID {
		t.Fatal("Failed disconnect Agent from module socket 2")
	}
	packet.SetAck()

	wg.Wait()
}

func TestMoveAgentToGroup(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket1 := proto.NewModule("test", groupID)
	if moduleSocket1 != nil {
		if !proto.AddModule(moduleSocket1) {
			t.Fatal("Failed add Module Socket object 1")
		}
	} else {
		t.Fatal("Failed initialize Module Socket object 1")
	}
	defer func() {
		if !proto.DelModule(moduleSocket1) {
			t.Fatal("Failed delete Module Socket object 1")
		}
	}()

	newGroupID := "56789012345678901234567890123456"
	moduleSocket2 := proto.NewModule("test", newGroupID)
	if moduleSocket2 != nil {
		if !proto.AddModule(moduleSocket2) {
			t.Fatal("Failed add Module Socket object 2")
		}
	} else {
		t.Fatal("Failed initialize Module Socket object 2")
	}
	defer func() {
		if !proto.DelModule(moduleSocket2) {
			t.Fatal("Failed delete Module Socket object 2")
		}
	}()

	var wg sync.WaitGroup
	syncMoveAgent := make(chan struct{})
	agentSocket := makeAgentSocket(proto)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if !proto.addAgent(ctx, agentSocket) {
			t.Error("Failed add Agent Socket object")
		}
		<-syncMoveAgent
		if !proto.delAgent(ctx, agentSocket) {
			t.Error("Failed delete Agent Socket object")
		}
	}()

	var packet *Packet
	packet = <-moduleSocket1.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentConnected ||
		packet.GetControlMsg().AgentInfo.ID != agentID ||
		packet.GetControlMsg().AgentInfo.GID != groupID {
		t.Fatal("Failed connect Agent to module socket 1")
	}
	packet.SetAck()

	proto.MoveAgent(ctx, agentID, newGroupID)

	packet = <-moduleSocket1.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentDisconnected ||
		packet.GetControlMsg().AgentInfo.ID != agentID ||
		packet.GetControlMsg().AgentInfo.GID != groupID {
		t.Fatal("Failed disconnect Agent from module socket 1")
	}
	packet.SetAck()

	packet = <-moduleSocket2.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentConnected ||
		packet.GetControlMsg().AgentInfo.ID != agentID ||
		packet.GetControlMsg().AgentInfo.GID != newGroupID {
		t.Fatal("Failed connect Agent to module socket 2")
	}
	packet.SetAck()

	syncMoveAgent <- struct{}{}

	packet = <-moduleSocket2.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentDisconnected ||
		packet.GetControlMsg().AgentInfo.ID != agentID ||
		packet.GetControlMsg().AgentInfo.GID != newGroupID {
		t.Fatal("Failed disconnect Agent from module socket 2")
	}
	packet.SetAck()

	wg.Wait()
}

func TestMoveAgentAndSendDataBetweenOnes(t *testing.T) {
	ctx := context.Background()
	var wg sync.WaitGroup
	testData := []byte("test data message")
	recvSyncChan1 := make(chan struct{})
	recvSyncChan2 := make(chan struct{})
	syncMoveAgent := make(chan struct{})
	syncQuitAgent := make(chan struct{})
	syncRunModule1 := make(chan struct{}, 1)
	syncRunModule2 := make(chan struct{}, 1)
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket1 := proto.NewModule("test", groupID)
	if moduleSocket1 != nil {
		if !proto.AddModule(moduleSocket1) {
			t.Fatal("Failed add Module Socket object 1")
		}
	} else {
		t.Fatal("Failed initialize Module Socket object 1")
	}
	defer func() {
		if !proto.DelModule(moduleSocket1) {
			t.Error("Failed delete Module Socket object 1")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		syncRunModule1 <- struct{}{}
		packetData, err := moduleSocket1.RecvDataFrom(ctx, agentToken, -1)
		if err != nil {
			t.Error("Failed receive data packet on module socket 1")
		}
		if !bytes.Equal(packetData.Data, testData) {
			t.Error("Failed compare of received data")
		}

		data := &Data{Data: testData}
		if err := moduleSocket1.SendDataTo(ctx, agentToken, data); err != nil {
			t.Error(err.Error())
		}
		recvSyncChan1 <- struct{}{}
	}()

	newGroupID := "56789012345678901234567890123456"
	moduleSocket2 := proto.NewModule("test", newGroupID)
	if moduleSocket2 != nil {
		if !proto.AddModule(moduleSocket2) {
			t.Fatal("Failed add Module Socket object 2")
		}
	} else {
		t.Fatal("Failed initialize Module Socket object 2")
	}
	defer func() {
		if !proto.DelModule(moduleSocket2) {
			t.Error("Failed delete Module Socket object 2")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		syncRunModule2 <- struct{}{}
		packetData, err := moduleSocket2.RecvDataFrom(ctx, agentToken, -1)
		if err != nil {
			t.Error("Failed receive data packet on module socket 2")
		}
		if !bytes.Equal(packetData.Data, testData) {
			t.Error("Failed compare of received data")
		}

		data := &Data{Data: testData}
		if err := moduleSocket2.SendDataTo(ctx, agentToken, data); err != nil {
			t.Error(err.Error())
		}
		recvSyncChan2 <- struct{}{}
	}()

	<-syncRunModule1
	<-syncRunModule2
	ctx, cancelCtx := context.WithCancel(ctx)
	fakeSocket := &FakeSocket{
		isConnected: true,
		RecvChan:    make(chan []byte),
		SendChan:    make(chan []byte),
		Ctx:         ctx,
		CancelCtx:   cancelCtx,
	}
	agentSocket := makeAgentSocket(proto)
	agentSocket.IConnection = fakeSocket

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		var packetData, recvData []byte
		addPacket := func(packet *Packet) {
			if err := proto.recvPacket(ctx, packet); err != nil {
				t.Error("Failed to receive packet:", err.Error())
			}
		}

		if !proto.addAgent(ctx, agentSocket) {
			t.Error("Failed add Agent Socket object")
		}

		addPacket(makePacketData(testData))
		packetData = <-fakeSocket.SendChan
		packetData, err = packEncrypter.Decrypt(packetData)
		if err != nil {
			t.Error("Failed to decrypt data packet from module 1:", err.Error())
		}
		recvData, err = parsePacketData(packetData)
		if err != nil {
			t.Error("Failed to receive data packet from module 1:", err.Error())
		}
		if !bytes.Equal(recvData, testData) {
			t.Error("Failed compare of received data")
		}

		<-syncMoveAgent

		addPacket(makePacketData(testData))
		packetData = <-fakeSocket.SendChan
		packetData, err = packEncrypter.Decrypt(packetData)
		if err != nil {
			t.Error("Failed to decrypt data packet from module 2:", err.Error())
		}
		recvData, err = parsePacketData(packetData)
		if err != nil {
			t.Error("Failed to receive data packet from module 2:", err.Error())
		}
		if !bytes.Equal(recvData, testData) {
			t.Error("Failed compare of received data")
		}

		<-syncQuitAgent

		if !proto.delAgent(ctx, agentSocket) {
			t.Error("Failed delete Agent Socket object")
		}
	}()

	var packet *Packet
	packet = <-moduleSocket1.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentConnected ||
		packet.GetControlMsg().AgentInfo.ID != agentID ||
		packet.GetControlMsg().AgentInfo.GID != groupID {
		t.Fatal("Failed connect Agent to module socket 1")
	}
	packet.SetAck()

	<-recvSyncChan1
	proto.MoveAgent(ctx, agentID, newGroupID)

	packet = <-moduleSocket1.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentDisconnected ||
		packet.GetControlMsg().AgentInfo.ID != agentID ||
		packet.GetControlMsg().AgentInfo.GID != groupID {
		t.Fatal("Failed disconnect Agent from module socket 1")
	}
	packet.SetAck()

	packet = <-moduleSocket2.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentConnected ||
		packet.GetControlMsg().AgentInfo.ID != agentID ||
		packet.GetControlMsg().AgentInfo.GID != newGroupID {
		t.Fatal("Failed connect Agent to module socket 2")
	}
	packet.SetAck()

	syncMoveAgent <- struct{}{}
	<-recvSyncChan2
	syncQuitAgent <- struct{}{}

	packet = <-moduleSocket2.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentDisconnected ||
		packet.GetControlMsg().AgentInfo.ID != agentID ||
		packet.GetControlMsg().AgentInfo.GID != newGroupID {
		t.Fatal("Failed disconnect Agent from module socket 2")
	}
	packet.SetAck()

	wg.Wait()
}

func TestFakeSocketSendFromAgentToModule(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket := proto.NewModule("test", groupID)
	if moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			t.Fatal("Failed add Module Socket object")
		}
	} else {
		t.Fatal("Failed initialize Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket) {
			t.Fatal("Failed delete Module Socket object")
		}
	}()

	var wg sync.WaitGroup
	numPackets := 5
	testData := []byte("test data message")
	actName := "test_action_name"
	textName := "test_text_name"
	fileName := "test_file_name.tmp"
	msgType := MTDebug
	agentSocket := makeAgentSocket(proto)
	recvSyncChan := make(chan struct{})
	quitSyncChan := make(chan struct{})
	ctx, cancelCtx := context.WithCancel(ctx)
	fakeSocket := &FakeSocket{
		isConnected: true,
		RecvChan:    make(chan []byte),
		SendChan:    make(chan []byte),
		Ctx:         ctx,
		CancelCtx:   cancelCtx,
	}
	agentSocket.IConnection = fakeSocket

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-recvSyncChan

		if !proto.addAgent(ctx, agentSocket) {
			t.Error("Failed add Agent Socket object")
		}
		defer func() {
			if !proto.delAgent(ctx, agentSocket) {
				t.Error("Failed delete Agent Socket object")
			}
		}()
		defer func() {
			if fakeSocket.Close(ctx) != nil {
				t.Error("Failed close Fake Socket object")
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			for ix := 0; ix < numPackets; ix++ {
				err := agentSocket.recvPacket(ctx)
				if err != nil && fakeSocket.isConnected {
					t.Error("Failed to receive packet:", err.Error())
				}
			}
		}()

		addPacket := func(packet *Packet) {
			packetData, err := packet.toBytesPB()
			if err != nil {
				t.Error("Failed to make packet:", err.Error())
			}

			packetData, err = packEncrypter.Encrypt(packetData)
			if err != nil {
				t.Error("Failed to encrypt packet:", err.Error())
			}
			fakeSocket.RecvChan <- packetData
		}

		addPacket(makePacketData(testData))
		addPacket(makePacketText(testData, textName))
		addPacket(makePacketFile(testData, "", fileName))
		addPacket(makePacketMsg(testData, msgType))
		addPacket(makePacketAction(testData, actName))

		<-quitSyncChan
	}()

	recvSyncChan <- struct{}{}
	var packet *Packet
	packet = <-moduleSocket.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentConnected || packet.GetControlMsg().AgentInfo.ID != agentID {
		t.Fatal("Failed connect Agent to proto")
	}
	packet.SetAck()

	packet = <-moduleSocket.GetReceiver()
	if !bytes.Equal(packet.GetData().Data, testData) {
		t.Fatal("Failed compare of received data")
	}
	packet.SetAck()
	packet = <-moduleSocket.GetReceiver()
	if !bytes.Equal(packet.GetText().Data, testData) || packet.GetText().Name != textName {
		t.Fatal("Failed compare of received text")
	}
	packet.SetAck()
	packet = <-moduleSocket.GetReceiver()
	if !bytes.Equal(packet.GetFile().Data, testData) || packet.GetFile().Name != fileName || packet.GetFile().Path != "" {
		t.Fatal("Failed compare of received file")
	}
	packet.SetAck()
	packet = <-moduleSocket.GetReceiver()
	if !bytes.Equal(packet.GetMsg().Data, testData) || packet.GetMsg().MType != msgType {
		t.Fatal("Failed compare of received message")
	}
	packet.SetAck()
	packet = <-moduleSocket.GetReceiver()
	if !bytes.Equal(packet.GetAction().Data, testData) || packet.GetAction().Name != actName {
		t.Fatal("Failed compare of received action")
	}
	packet.SetAck()

	quitSyncChan <- struct{}{}
	packet = <-moduleSocket.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentDisconnected || packet.GetControlMsg().AgentInfo.ID != agentID {
		t.Fatal("Failed disconnect Agent from proto")
	}
	packet.SetAck()

	wg.Wait()
}

func TestFakeSocketSendFromModuleToAgent(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket := proto.NewModule("test", groupID)
	if moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			t.Fatal("Failed add Module Socket object")
		}
	} else {
		t.Fatal("Failed initialize Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket) {
			t.Fatal("Failed delete Module Socket object")
		}
	}()

	var wg sync.WaitGroup
	testData := []byte("test data message")
	actName := "test_action_name"
	textName := "test_text_name"
	fileName := "test_file_name.tmp"
	msgType := MTDebug
	agentSocket := makeAgentSocket(proto)
	ctx, cancelCtx := context.WithCancel(ctx)
	fakeSocket := &FakeSocket{
		isConnected: true,
		RecvChan:    make(chan []byte),
		SendChan:    make(chan []byte),
		Ctx:         ctx,
		CancelCtx:   cancelCtx,
	}
	agentSocket.IConnection = fakeSocket

	wg.Add(1)
	go func() {
		defer wg.Done()
		if !proto.addAgent(ctx, agentSocket) {
			t.Error("Failed add Agent Socket object")
		}
		defer func() {
			if !proto.delAgent(ctx, agentSocket) {
				t.Error("Failed delete Agent Socket object")
			}
		}()
		defer func() {
			if fakeSocket.Close(ctx) != nil {
				t.Error("Failed close Fake Socket object")
			}
		}()

		var (
			err        error
			packetData []byte
		)
		decrypt := func(data []byte) []byte {
			decdata, err := packEncrypter.Decrypt(data)
			if err != nil {
				t.Error("Failed to decrypt data packet:", err.Error())
			}
			return decdata
		}
		packetData = <-fakeSocket.SendChan
		recvData, err := parsePacketData(decrypt(packetData))
		if err != nil {
			t.Error("Failed to receive data packet:", err.Error())
		}
		if !bytes.Equal(recvData, testData) {
			t.Error("Failed compare of received data")
		}

		packetData = <-fakeSocket.SendChan
		recvData, recvName, err := parsePacketText(decrypt(packetData))
		if err != nil {
			t.Error("Failed to receive text packet:", err.Error())
		}
		if !bytes.Equal(recvData, testData) || recvName != textName {
			t.Error("Failed compare of received text")
		}

		packetData = <-fakeSocket.SendChan
		recvData, recvPath, recvName, err := parsePacketFile(decrypt(packetData))
		if err != nil {
			t.Error("Failed to receive file packet:", err.Error())
		}
		if !bytes.Equal(recvData, testData) || recvName != fileName || recvPath != "" {
			t.Error("Failed compare of received file")
		}

		packetData = <-fakeSocket.SendChan
		recvData, recvMsgType, err := parsePacketMsg(decrypt(packetData))
		if err != nil {
			t.Error("Failed to receive message packet:", err.Error())
		}
		if !bytes.Equal(recvData, testData) || recvMsgType != msgType {
			t.Error("Failed compare of received message")
		}

		packetData = <-fakeSocket.SendChan
		recvData, recvName, err = parsePacketAction(decrypt(packetData))
		if err != nil {
			t.Error("Failed to receive action packet:", err.Error())
		}
		if !bytes.Equal(recvData, testData) || recvName != actName {
			t.Error("Failed compare of received action")
		}
	}()

	var packet *Packet
	packet = <-moduleSocket.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentConnected || packet.GetControlMsg().AgentInfo.ID != agentID {
		t.Fatal("Failed connect Agent to proto")
	}
	packet.SetAck()

	data := &Data{Data: testData}
	if err := moduleSocket.SendDataTo(ctx, agentToken, data); err != nil {
		t.Fatal(err.Error())
	}
	text := &Text{Data: testData, Name: textName}
	if err := moduleSocket.SendTextTo(ctx, agentToken, text); err != nil {
		t.Fatal(err.Error())
	}
	file := &File{Data: testData, Name: fileName, Path: ""}
	if err := moduleSocket.SendFileTo(ctx, agentToken, file); err != nil {
		t.Fatal(err.Error())
	}
	msg := &Msg{Data: testData, MType: msgType}
	if err := moduleSocket.SendMsgTo(ctx, agentToken, msg); err != nil {
		t.Fatal(err.Error())
	}
	act := &Action{Data: testData, Name: actName}
	if err := moduleSocket.SendActionTo(ctx, agentToken, act); err != nil {
		t.Fatal(err.Error())
	}

	packet = <-moduleSocket.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentDisconnected || packet.GetControlMsg().AgentInfo.ID != agentID {
		t.Fatal("Failed disconnect Agent from proto")
	}
	packet.SetAck()

	wg.Wait()
}

func TestUsingProtoStatsViaFakeSocket(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket := proto.NewModule("test", groupID)
	if moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			t.Fatal("Failed add Module Socket object")
		}
	} else {
		t.Fatal("Failed initialize Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket) {
			t.Fatal("Failed delete Module Socket object")
		}
	}()

	var wg sync.WaitGroup
	numPackets := 100
	testData := []byte("test data message")
	agentSocket := makeAgentSocket(proto)
	ctx, cancelCtx := context.WithCancel(ctx)
	fakeSocket := &FakeSocket{
		isConnected: true,
		RecvChan:    make(chan []byte),
		SendChan:    make(chan []byte),
		Ctx:         ctx,
		CancelCtx:   cancelCtx,
	}
	agentSocket.IConnection = fakeSocket

	wg.Add(1)
	go func() {
		defer wg.Done()
		if !proto.addAgent(ctx, agentSocket) {
			t.Error("Failed add Agent Socket object")
		}
		defer func() {
			if !proto.delAgent(ctx, agentSocket) {
				t.Error("Failed delete Agent Socket object")
			}
		}()
		defer func() {
			if fakeSocket.Close(ctx) != nil {
				t.Error("Failed close Fake Socket object")
			}
		}()

		decrypt := func(data []byte) []byte {
			decdata, err := packEncrypter.Decrypt(data)
			if err != nil {
				t.Error("Failed to decrypt data packet:", err.Error())
			}
			return decdata
		}
		for i := 0; i < numPackets; i++ {
			packetData := <-fakeSocket.SendChan
			recvData, err := parsePacketData(decrypt(packetData))
			if err != nil {
				t.Error("Failed to receive data packet:", err.Error())
			}
			if !bytes.Equal(recvData, testData) {
				t.Error("Failed compare of received data")
			}
		}
	}()

	var packet *Packet
	packet = <-moduleSocket.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentConnected || packet.GetControlMsg().AgentInfo.ID != agentID {
		t.Fatal("Failed connect Agent to proto")
	}
	packet.SetAck()

	for i := 0; i < numPackets; i++ {
		data := &Data{Data: testData}
		if err := moduleSocket.SendDataTo(ctx, agentToken, data); err != nil {
			t.Fatal(err.Error())
		}
	}

	packet = <-moduleSocket.GetReceiver()
	if packet.GetControlMsg().MsgType != AgentDisconnected || packet.GetControlMsg().AgentInfo.ID != agentID {
		t.Fatal("Failed disconnect Agent from proto")
	}
	packet.SetAck()

	wg.Wait()

	stats, err := proto.DumpStats()
	if err != nil {
		t.Fatalf("Failed to get dump stats: %v", err)
	}
	if val, ok := stats["proto_send_num_packets"]; !ok || val != float64(numPackets) {
		t.Fatal("Failed to compare of sent packets number")
	}
	if val, ok := stats["proto_send_payload_bytes"]; !ok || val != 171*float64(numPackets) {
		t.Fatal("Failed to compare of sent payload bytes")
	}
	if val, ok := stats["proto_send_net_bytes"]; !ok || val != 95*float64(numPackets) {
		t.Fatal("Failed to compare of sent bytes via network socket")
	}
}

func TestIMCSendRecvPackets(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	defer proto.Close(ctx)

	module1Token := proto.MakeIMCToken("test1", groupID)
	moduleSocket1 := proto.NewModule("test1", groupID)
	if moduleSocket1 != nil {
		if !proto.AddModule(moduleSocket1) {
			t.Fatal("Failed add first Module Socket object")
		}
	} else {
		t.Fatal("Failed initialize first Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket1) {
			t.Fatal("Failed delete first Module Socket object")
		}
	}()

	module2Token := proto.MakeIMCToken("test2", groupID)
	moduleSocket2 := proto.NewModule("test2", groupID)
	if moduleSocket2 != nil {
		if !proto.AddModule(moduleSocket2) {
			t.Fatal("Failed add second Module Socket object")
		}
	} else {
		t.Fatal("Failed initialize second Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket2) {
			t.Fatal("Failed delete second Module Socket object")
		}
	}()

	var wg sync.WaitGroup
	testData := []byte("test data message")
	actName := "test_action_name"
	textName := "test_text_name"
	fileName := "test_file_name.tmp"
	msgType := MTDebug
	recvSyncChan := make(chan struct{})
	sendSyncChan := make(chan struct{})

	batchReceive := func(ms IModuleSocket, src, dst string) {
		checkRoutingInfo := func(packet *Packet) {
			if packet.Src != src || packet.Dst != dst {
				t.Fatal("Failed routing info of received packet on " + ms.GetName())
			}
		}

		var packet *Packet
		packet = <-ms.GetReceiver()
		if !bytes.Equal(packet.GetData().Data, testData) {
			t.Fatal("Failed compare of received data on " + ms.GetName())
		}
		packet.SetAck()
		checkRoutingInfo(packet)
		packet = <-ms.GetReceiver()
		if !bytes.Equal(packet.GetText().Data, testData) || packet.GetText().Name != textName {
			t.Fatal("Failed compare of received text on " + ms.GetName())
		}
		packet.SetAck()
		checkRoutingInfo(packet)
		packet = <-ms.GetReceiver()
		if !checkFileData(packet.GetFile().Path, testData) || packet.GetFile().Name != fileName || packet.GetFile().Data != nil {
			t.Fatal("Failed compare of received file on " + ms.GetName())
		}
		packet.SetAck()
		checkRoutingInfo(packet)
		packet = <-ms.GetReceiver()
		if !bytes.Equal(packet.GetMsg().Data, testData) || packet.GetMsg().MType != msgType {
			t.Fatal("Failed compare of received message on " + ms.GetName())
		}
		packet.SetAck()
		checkRoutingInfo(packet)
		packet = <-ms.GetReceiver()
		if !bytes.Equal(packet.GetAction().Data, testData) || packet.GetAction().Name != actName {
			t.Fatal("Failed compare of received action on " + ms.GetName())
		}
		packet.SetAck()
		checkRoutingInfo(packet)
	}

	batchSend := func(ms IModuleSocket, dst string) {
		if err := ms.SendDataTo(ctx, dst, &Data{Data: testData}); err != nil {
			t.Fatal("Failed send data packet to modude via imc:", err.Error())
		}
		if err := ms.SendTextTo(ctx, dst, &Text{Data: testData, Name: textName}); err != nil {
			t.Fatal("Failed send text packet to modude via imc:", err.Error())
		}
		if err := ms.SendFileTo(ctx, dst, &File{Data: testData, Name: fileName, Path: ""}); err != nil {
			t.Fatal("Failed send file packet to modude via imc:", err.Error())
		}
		if err := ms.SendMsgTo(ctx, dst, &Msg{Data: testData, MType: msgType}); err != nil {
			t.Fatal("Failed send message packet to modude via imc:", err.Error())
		}
		if err := ms.SendActionTo(ctx, dst, &Action{Data: testData, Name: actName}); err != nil {
			t.Fatal("Failed send action packet to modude via imc:", err.Error())
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		sendSyncChan <- struct{}{}
		batchReceive(moduleSocket1, module2Token, module1Token)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-recvSyncChan
		batchSend(moduleSocket1, module2Token)
	}()

	recvSyncChan <- struct{}{}
	batchReceive(moduleSocket2, module1Token, module2Token)
	<-sendSyncChan
	batchSend(moduleSocket2, module1Token)

	wg.Wait()
}

func TestIMCSendRecvPacketsToTopic(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	defer proto.Close(ctx)

	topicName := "test_topic"
	topicToken := proto.MakeIMCTopic(topicName, groupID)
	module1Token := proto.MakeIMCToken("test1", groupID)
	moduleSocket1 := proto.NewModule("test1", groupID)
	if moduleSocket1 != nil {
		if !proto.AddModule(moduleSocket1) {
			t.Fatal("failed to add sender Module Socket object")
		}
	} else {
		t.Fatal("failed to initialize sender Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket1) {
			t.Fatal("failed to delete sender Module Socket object")
		}
	}()

	receivers := make([]IModuleSocket, 0)
	for idx := 0; idx < 5; idx++ {
		moduleID := fmt.Sprintf("test%d", idx+2)
		moduleSocket := proto.NewModule(moduleID, groupID)
		if moduleSocket != nil {
			if !proto.AddModule(moduleSocket) {
				t.Fatalf("failed to add receiver[%d] Module Socket object", idx)
			}
		} else {
			t.Fatalf("failed to initialize receiver[%d] Module Socket object", idx)
		}
		defer func() {
			if !proto.DelModule(moduleSocket) {
				t.Fatalf("failed to delete receiver[%d] Module Socket object", idx)
			}
		}()
		if !moduleSocket.SubscribeIMCToTopic(topicName, groupID, moduleSocket.GetIMCToken()) {
			t.Fatalf("failed to subscribe receiver[%d] Module Socket object to topic", idx)
		}
		receivers = append(receivers, moduleSocket)
	}

	var wg sync.WaitGroup
	testData := []byte("test data message")
	actName := "test_action_name"
	textName := "test_text_name"
	fileName := "test_file_name.tmp"
	msgType := MTDebug
	recvSyncChan := make(chan struct{})

	batchReceive := func(ms IModuleSocket, src, dst string) {
		checkRoutingInfo := func(packet *Packet) {
			if packet.Src != src || packet.Dst != dst {
				t.Fatal("failed to routing info of received packet on " + ms.GetName())
			}
		}

		var packet *Packet
		packet = <-ms.GetReceiver()
		if !bytes.Equal(packet.GetData().Data, testData) {
			t.Fatal("failed to compare of received data on " + ms.GetName())
		}
		packet.SetAck()
		checkRoutingInfo(packet)
		packet = <-ms.GetReceiver()
		if !bytes.Equal(packet.GetText().Data, testData) || packet.GetText().Name != textName {
			t.Fatal("failed to compare of received text on " + ms.GetName())
		}
		packet.SetAck()
		checkRoutingInfo(packet)
		packet = <-ms.GetReceiver()
		if !checkFileData(packet.GetFile().Path, testData) || packet.GetFile().Name != fileName || packet.GetFile().Data != nil {
			t.Fatal("failed to compare of received file on " + ms.GetName())
		}
		packet.SetAck()
		checkRoutingInfo(packet)
		packet = <-ms.GetReceiver()
		if !bytes.Equal(packet.GetMsg().Data, testData) || packet.GetMsg().MType != msgType {
			t.Fatal("failed to compare of received message on " + ms.GetName())
		}
		packet.SetAck()
		checkRoutingInfo(packet)
		packet = <-ms.GetReceiver()
		if !bytes.Equal(packet.GetAction().Data, testData) || packet.GetAction().Name != actName {
			t.Fatal("failed to compare of received action on " + ms.GetName())
		}
		packet.SetAck()
		checkRoutingInfo(packet)
	}

	batchSend := func(ms IModuleSocket, dst string) {
		if err := ms.SendDataTo(ctx, dst, &Data{Data: testData}); err != nil {
			t.Fatal("failed to send data packet to modude via imc:", err.Error())
		}
		if err := ms.SendTextTo(ctx, dst, &Text{Data: testData, Name: textName}); err != nil {
			t.Fatal("failed to send text packet to modude via imc:", err.Error())
		}
		if err := ms.SendFileTo(ctx, dst, &File{Data: testData, Name: fileName, Path: ""}); err != nil {
			t.Fatal("failed to send file packet to modude via imc:", err.Error())
		}
		if err := ms.SendMsgTo(ctx, dst, &Msg{Data: testData, MType: msgType}); err != nil {
			t.Fatal("failed to send message packet to modude via imc:", err.Error())
		}
		if err := ms.SendActionTo(ctx, dst, &Action{Data: testData, Name: actName}); err != nil {
			t.Fatal("failed to send action packet to modude via imc:", err.Error())
		}
	}

	for _, mod := range receivers {
		wg.Add(1)
		go func(module IModuleSocket) {
			defer wg.Done()
			<-recvSyncChan
			batchReceive(module, module1Token, topicToken)
		}(mod)
	}

	close(recvSyncChan)
	batchSend(moduleSocket1, topicToken)

	wg.Wait()
}

func TestIMCSendToUnknownDestination(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket := proto.NewModule("test", groupID)
	if moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			t.Fatal("failed to add Module Socket object")
		}
	} else {
		t.Fatal("failed to initialize Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket) {
			t.Fatal("failed to delete Module Socket object")
		}
	}()

	type destination struct {
		token  string
		expect error
	}
	dsts := []destination{
		{proto.MakeIMCToken("unknown", groupID), ErrDstUnreachable},
		{proto.MakeIMCTopic("unknown", groupID), ErrTopicUnreachable},
	}
	testData := []byte("test data message")
	textName := "test_text_name"
	fileName := "test_file_name.tmp"
	msgType := MTDebug

	for _, d := range dsts {
		dst := d.token
		err := moduleSocket.SendDataTo(ctx, dst, &Data{Data: testData})
		if !errors.Is(err, d.expect) {
			t.Fatal("failed to send data packet to modude via imc", err)
		}
		err = moduleSocket.SendTextTo(ctx, dst, &Text{Data: testData, Name: textName})
		if !errors.Is(err, d.expect) {
			t.Fatal("failed to send text packet to modude via imc", err)
		}
		err = moduleSocket.SendFileTo(ctx, dst, &File{Data: testData, Name: fileName, Path: ""})
		if !errors.Is(err, d.expect) {
			t.Fatal("failed to send file packet to modude via imc", err)
		}
		err = moduleSocket.SendMsgTo(ctx, dst, &Msg{Data: testData, MType: msgType})
		if !errors.Is(err, d.expect) {
			t.Fatal("failed to send message packet to modude via imc", err)
		}
	}
}

func TestIMCTopicsAPI(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket := proto.NewModule("test", groupID)
	if moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			t.Fatal("failed to add Module Socket object")
		}
	} else {
		t.Fatal("failed to initialize Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket) {
			t.Fatal("failed to delete Module Socket object")
		}
	}()

	imcToken := moduleSocket.GetIMCToken()
	testData := []byte("test data message")
	for idx := 0; idx < 10000; idx++ {
		// On publisher side: should make topic token
		topicName := fmt.Sprintf("test_topic%d", idx)
		topicToken := moduleSocket.MakeIMCTopic(topicName, groupID)
		if !strings.HasPrefix(topicToken, "ffff7777") {
			t.Fatal("failed to prepare topic token")
		}

		// On publisher side: check topic availability
		if tokenInfo := moduleSocket.GetIMCTopic(topicToken); tokenInfo != nil {
			// This topic doesn't contain any subscribers and doesn't register in VXProto
			t.Fatal("unregistered topic must return nil as an info")
		}

		// On publisher side: test sending packet to unavailable topic
		err := moduleSocket.SendDataTo(ctx, topicToken, &Data{Data: testData})
		if !errors.Is(err, ErrTopicUnreachable) {
			// Sending a packet to unavailable topic must raise ErrTopicUnreachable error
			t.Fatal("send data call returned unexpected error", err)
		}

		// On subscriber side: subscribe to the topic and add an IMC token to the list
		if !moduleSocket.SubscribeIMCToTopic(topicName, groupID, imcToken) {
			// Subscription to an unregistered topic should succeed
			t.Fatal("unregistered topic must be available to subscribing")
		}

		// On publisher side: double check that topic is available
		if topicInfoR := moduleSocket.GetIMCTopic(topicToken); topicInfoR == nil {
			// This topic must be registered in VXProto after previosly SubscribeIMC* call
			t.Fatal("registered topic must return info about itself")
		} else if topicInfoR.GetName() != topicName || topicInfoR.GetGroupID() != groupID {
			// This topic contains mismatch name or group ID
			t.Fatal("registered topic must return correct info about itself")
		} else if !stringInSlice(imcToken, topicInfoR.GetSubscriptions()) {
			// Current IMC token must contain in the topic subscriptions list
			t.Fatal("registered topic must contain IMC token which was added")
		}

		// On subscriber side: run routine to receive a packet
		go func() {
			src, data, err := moduleSocket.RecvData(ctx, 1000)
			if err != nil || src != imcToken || !bytes.Equal(data.Data, testData) {
				t.Error("error reading packet from topic")
			}
		}()

		// On publisher side: test sending packet to a available topic
		if err := moduleSocket.SendDataTo(ctx, topicToken, &Data{Data: testData}); err != nil {
			// Sending packets to a registered topics should not raise any errors
			t.Fatal("send data call returned an error", err)
		}

		// On subscriber side: unsubscribe from the topic and delete an IMC token from the list
		if !moduleSocket.UnsubscribeIMCFromTopic(topicName, groupID, imcToken) {
			t.Fatal("registered topic must be available for unsubscribing")
		}
	}

	if len(proto.GetIMCTopics()) != 0 {
		t.Fatal("registered topics list must be empty once everyone got unsubscribed")
	}

	// Check an API when there are nothing to unsubscribe
	if !moduleSocket.UnsubscribeIMCFromAllTopics(imcToken) {
		t.Fatal("failed to unsubscribe while there should be no subscriptions")
	}
}

func TestIMCSendRecvLargeFiles(t *testing.T) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		t.Fatal(err)
	}
	defer proto.Close(ctx)

	module1Token := proto.MakeIMCToken("test1", groupID)
	moduleSocket1 := proto.NewModule("test1", groupID)
	if moduleSocket1 != nil {
		if !proto.AddModule(moduleSocket1) {
			t.Fatal("Failed add first Module Socket object")
		}
	} else {
		t.Fatal("Failed initialize first Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket1) {
			t.Fatal("Failed delete first Module Socket object")
		}
	}()

	module2Token := proto.MakeIMCToken("test2", groupID)
	moduleSocket2 := proto.NewModule("test2", groupID)
	if moduleSocket2 != nil {
		if !proto.AddModule(moduleSocket2) {
			t.Fatal("Failed add second Module Socket object")
		}
	} else {
		t.Fatal("Failed initialize second Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket2) {
			t.Fatal("Failed delete second Module Socket object")
		}
	}()

	var wg sync.WaitGroup
	testData := bytes.Repeat([]byte("test data message"), 100000)
	fileName := "test_file_name.tmp"

	batchReceive := func(ms IModuleSocket, src, dst string) {
		checkRoutingInfo := func(packet *Packet) {
			if packet.Src != src || packet.Dst != dst {
				t.Fatal("Failed routing info of received packet on " + ms.GetName())
			}
		}

		packet := <-ms.GetReceiver()
		packet.SetAck()
		if !checkFileData(packet.GetFile().Path, testData) || packet.GetFile().Name != fileName || packet.GetFile().Data != nil {
			t.Fatal("Failed compare of received file (from memory) on " + ms.GetName())
		}
		checkRoutingInfo(packet)

		packet = <-ms.GetReceiver()
		packet.SetAck()
		if !checkFileData(packet.GetFile().Path, testData) || packet.GetFile().Name != fileName || packet.GetFile().Data != nil {
			t.Fatal("Failed compare of received file (from FS) on " + ms.GetName())
		}
		checkRoutingInfo(packet)
	}

	batchSend := func(ms IModuleSocket, dst string) {
		if err := ms.SendFileTo(ctx, dst, &File{Data: testData, Name: fileName, Path: ""}); err != nil {
			t.Fatal("Failed send file packet (from memory) to modude via imc:", err.Error())
		}
		tempFile := filepath.Join(os.TempDir(), "vxlargefile.tmp")
		if err := ioutil.WriteFile(tempFile, testData, 0666); err != nil {
			t.Fatal("Failed dump large file to FS:", err.Error())
		}
		for i := 0; i < 50 && !checkFileData(tempFile, testData); i++ {
			time.Sleep(100 * time.Millisecond)
		}
		if err := ms.SendFileTo(ctx, dst, &File{Name: fileName, Path: tempFile}); err != nil {
			t.Fatal("Failed send file packet (from FS) to modude via imc:", err.Error())
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		batchReceive(moduleSocket1, module2Token, module1Token)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		batchSend(moduleSocket1, module2Token)
	}()

	batchReceive(moduleSocket2, module1Token, module2Token)
	batchSend(moduleSocket2, module1Token)

	wg.Wait()
}

func BenchmarkStabCreatingProto(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proto, err := getVXProto()
		if err != nil {
			b.Fatal(err)
		}
		if err := proto.Close(ctx); err != nil {
			b.Fatal(err.Error())
		}
	}
}

func BenchmarkStabCreatingModule(b *testing.B) {
	ctx := context.Background()
	proto, err := getVXProto()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			modName := randString(10)
			if moduleSocket := proto.NewModule(modName, groupID); moduleSocket != nil {
				if !proto.AddModule(moduleSocket) {
					b.Fatal("Failed add Module Socket object")
				}
				if !proto.DelModule(moduleSocket) {
					b.Fatal("Failed delete Module Socket object")
				}
			} else {
				b.Fatal("Failed initialize Module Socket object")
			}
		}
	})

	if err := proto.Close(ctx); err != nil {
		b.Fatal(err.Error())
	}
}

func BenchmarkStabLinkAgentToModule(b *testing.B) {
	ctx := context.Background()
	var cntAgents, cntCon, cntDis int64
	var wg sync.WaitGroup
	quit := make(chan struct{})
	proto, err := getVXProto()
	if err != nil {
		b.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket := proto.NewModule("test", groupID)
	if moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			b.Fatal("Failed add Module Socket object")
		}
	} else {
		b.Fatal("Failed initialize Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket) {
			b.Fatal("Failed delete Module Socket object")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-quit:
				return
			case packet := <-moduleSocket.GetReceiver():
				packet.SetAck()
				if packet.GetControlMsg().AgentInfo.ID != agentID {
					b.Error("Failed to get Agent ID from packet proto")
				}
				if packet.GetControlMsg().MsgType == AgentConnected {
					atomic.AddInt64(&cntCon, 1)
					continue
				}
				if packet.GetControlMsg().MsgType == AgentDisconnected {
					atomic.AddInt64(&cntDis, 1)
					continue
				}
				b.Error("Failed to get Message Type from packet proto")
			}
		}
	}()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			agentSocket := makeAgentSocket(proto)
			agentSocket.src = randString(20)
			agentSocket.auth.resp.Stoken = &agentSocket.src
			agentToken := randString(20)
			agentSocket.auth.req.Atoken = &agentToken
			agentSocket.auth.resp.Atoken = &agentToken
			if !proto.addAgent(ctx, agentSocket) {
				b.Fatal("Failed add Agent Socket object")
			}
			if !proto.delAgent(ctx, agentSocket) {
				b.Fatal("Failed delete Agent Socket object")
			}
			atomic.AddInt64(&cntAgents, 1)
		}
	})
	b.StopTimer()

	for atomic.LoadInt64(&cntCon) != atomic.LoadInt64(&cntAgents) {
		runtime.Gosched()
	}
	for atomic.LoadInt64(&cntDis) != atomic.LoadInt64(&cntAgents) {
		runtime.Gosched()
	}

	quit <- struct{}{}
	wg.Wait()
}

func BenchmarkStabAPISendFromAgentToModule(b *testing.B) {
	ctx := context.Background()
	var cntTotal, cntRecv int64
	var wg sync.WaitGroup
	quit := make(chan struct{})
	proto, err := getVXProto()
	if err != nil {
		b.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket := proto.NewModule("test", groupID)
	if moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			b.Fatal("Failed add Module Socket object")
		}
	} else {
		b.Fatal("Failed initialize Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket) {
			b.Fatal("Failed delete Module Socket object")
		}
	}()

	testData := []byte("test data message")
	testDataLen := int64(len(testData))
	agentSocket := makeAgentSocket(proto)
	addPacket := func(packet *Packet) {
		if err := proto.recvPacket(ctx, packet); err != nil {
			b.Fatal("Failed to add packet to the queue:", err.Error())
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-quit:
				return
			case packet := <-moduleSocket.GetReceiver():
				packet.SetAck()
				switch packet.PType {
				case PTData:
					if !bytes.Equal(packet.GetData().Data, testData) {
						b.Error("Failed compare of received data")
					}
					atomic.AddInt64(&cntRecv, 1)
				case PTControl:
					if packet.GetControlMsg().AgentInfo.ID != agentID {
						b.Error("Failed to get Agent ID from packet proto")
					}
					if packet.GetControlMsg().MsgType == AgentConnected {
						continue
					}
					if packet.GetControlMsg().MsgType == AgentDisconnected {
						continue
					}
					b.Error("Failed to get Message Type from packet proto")
				}
			}
		}
	}()

	if !proto.addAgent(ctx, agentSocket) {
		b.Fatal("Failed add Agent Socket object")
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			addPacket(makePacketData(testData))
			atomic.AddInt64(&cntTotal, 1)
			b.SetBytes(testDataLen)
		}
	})
	b.StopTimer()
	if !proto.delAgent(ctx, agentSocket) {
		b.Fatal("Failed delete Agent Socket object")
	}

	for atomic.LoadInt64(&cntRecv) != atomic.LoadInt64(&cntTotal) {
		runtime.Gosched()
	}

	quit <- struct{}{}
	wg.Wait()
}

func BenchmarkStabConnectAndSendFromAgentToModule(b *testing.B) {
	ctx := context.Background()
	var cntTotal, cntRecv, cntAgents, cntCon, cntDis int64
	var wg sync.WaitGroup
	quit := make(chan struct{})
	proto, err := getVXProto()
	if err != nil {
		b.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket := proto.NewModule("test", groupID)
	if moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			b.Fatal("Failed add Module Socket object")
		}
	} else {
		b.Fatal("Failed initialize Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket) {
			b.Fatal("Failed delete Module Socket object")
		}
	}()

	testData := []byte("test data message")
	testDataLen := int64(len(testData))
	addPacket := func(packet *Packet) {
		if err := proto.recvPacket(ctx, packet); err != nil {
			b.Fatal("Failed to add packet to the queue:", err.Error())
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-quit:
				return
			case packet := <-moduleSocket.GetReceiver():
				packet.SetAck()
				switch packet.PType {
				case PTData:
					if !bytes.Equal(packet.GetData().Data, testData) {
						b.Error("Failed compare of received data")
					}
					atomic.AddInt64(&cntRecv, 1)
				case PTControl:
					if packet.GetControlMsg().AgentInfo.ID != agentID {
						b.Error("Failed to get Agent ID from packet proto")
					}
					if packet.GetControlMsg().MsgType == AgentConnected {
						atomic.AddInt64(&cntCon, 1)
						continue
					}
					if packet.GetControlMsg().MsgType == AgentDisconnected {
						atomic.AddInt64(&cntDis, 1)
						continue
					}
					b.Error("Failed to get Message Type from packet proto")
				}
			}
		}
	}()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		agentSocket := makeAgentSocket(proto)
		agentSocket.src = randString(20)
		agentSocket.auth.resp.Stoken = &agentSocket.src
		agentToken := randString(20)
		agentSocket.auth.req.Atoken = &agentToken
		agentSocket.auth.resp.Atoken = &agentToken
		if !proto.addAgent(ctx, agentSocket) {
			b.Fatal("Failed add Agent Socket object")
		}
		for pb.Next() {
			packet := makePacketData(testData)
			packet.Src = agentToken
			packet.Dst = agentSocket.src
			addPacket(packet)
			atomic.AddInt64(&cntTotal, 1)
			b.SetBytes(testDataLen)
		}
		if !proto.delAgent(ctx, agentSocket) {
			b.Fatal("Failed delete Agent Socket object")
		}
		atomic.AddInt64(&cntAgents, 1)
	})
	b.StopTimer()

	for atomic.LoadInt64(&cntRecv) != atomic.LoadInt64(&cntTotal) {
		runtime.Gosched()
	}
	for atomic.LoadInt64(&cntCon) != atomic.LoadInt64(&cntAgents) {
		runtime.Gosched()
	}
	for atomic.LoadInt64(&cntDis) != atomic.LoadInt64(&cntAgents) {
		runtime.Gosched()
	}

	quit <- struct{}{}
	wg.Wait()
}

func BenchmarkStabFakeSocketSendFromAgentToModule(b *testing.B) {
	ctx := context.Background()
	var cntTotal, cntAgentRecv, cntModuleRecv int64
	var wg sync.WaitGroup
	quit := make(chan struct{})
	proto, err := getVXProto()
	if err != nil {
		b.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket := proto.NewModule("test", groupID)
	if moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			b.Fatal("Failed add Module Socket object")
		}
	} else {
		b.Fatal("Failed initialize Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket) {
			b.Fatal("Failed delete Module Socket object")
		}
	}()

	testData := []byte("test data message")
	testDataLen := int64(len(testData))
	agentSocket := makeAgentSocket(proto)
	ctx, cancelCtx := context.WithCancel(ctx)
	fakeSocket := &FakeSocket{
		isConnected: true,
		RecvChan:    make(chan []byte),
		SendChan:    make(chan []byte),
		Ctx:         ctx,
		CancelCtx:   cancelCtx,
	}
	agentSocket.IConnection = fakeSocket
	addPacket := func(packet *Packet) {
		packetData, err := packet.toBytesPB()
		if err != nil {
			b.Fatal("Failed to make packet:", err.Error())
		}

		packetData, err = packEncrypter.Encrypt(packetData)
		if err != nil {
			b.Error("Failed to encrypt packet:", err.Error())
		}
		fakeSocket.RecvChan <- packetData
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for atomic.LoadInt64(&cntAgentRecv) != atomic.LoadInt64(&cntTotal) || fakeSocket.isConnected {
			err := agentSocket.recvPacket(ctx)
			if err != nil {
				if fakeSocket.isConnected {
					b.Error("Failed to receive packet:", err.Error())
				} else {
					return
				}
			} else {
				atomic.AddInt64(&cntAgentRecv, 1)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-quit:
				return
			case packet := <-moduleSocket.GetReceiver():
				packet.SetAck()
				switch packet.PType {
				case PTData:
					if !bytes.Equal(packet.GetData().Data, testData) {
						b.Error("Failed compare of received data")
					}
					atomic.AddInt64(&cntModuleRecv, 1)
				case PTControl:
					if packet.GetControlMsg().AgentInfo.ID != agentID {
						b.Error("Failed to get Agent ID from packet proto")
					}
					if packet.GetControlMsg().MsgType == AgentConnected {
						continue
					}
					if packet.GetControlMsg().MsgType == AgentDisconnected {
						continue
					}
					b.Error("Failed to get Message Type from packet proto")
				default:
					b.Error("Failed to parse packet type")
				}
			}
		}
	}()

	if !proto.addAgent(ctx, agentSocket) {
		b.Fatal("Failed add Agent Socket object")
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			addPacket(makePacketData(testData))
			atomic.AddInt64(&cntTotal, 1)
			b.SetBytes(testDataLen)
		}
	})
	b.StopTimer()

	for atomic.LoadInt64(&cntModuleRecv) != atomic.LoadInt64(&cntTotal) {
		runtime.Gosched()
	}
	for atomic.LoadInt64(&cntAgentRecv) != atomic.LoadInt64(&cntTotal) {
		runtime.Gosched()
	}

	if !proto.delAgent(ctx, agentSocket) {
		b.Fatal("Failed delete Agent Socket object")
	}
	if !errors.Is(fakeSocket.Close(ctx), websocket.ErrCloseSent) {
		b.Error("Fake Socket object must be closed on delete agent api")
	}

	quit <- struct{}{}
	wg.Wait()
}

func BenchmarkStabFakeSocketSendFromModuleToAgent(b *testing.B) {
	ctx := context.Background()
	var cntTotal, cntRecv int64
	var wg sync.WaitGroup
	quit := make(chan struct{})
	proto, err := getVXProto()
	if err != nil {
		b.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket := proto.NewModule("test", groupID)
	if moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			b.Fatal("Failed add Module Socket object")
		}
	} else {
		b.Fatal("Failed initialize Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket) {
			b.Fatal("Failed delete Module Socket object")
		}
	}()

	testData := []byte("test data message")
	testDataLen := int64(len(testData))
	agentSocket := makeAgentSocket(proto)
	ctx, cancelCtx := context.WithCancel(ctx)
	fakeSocket := &FakeSocket{
		isConnected: true,
		RecvChan:    make(chan []byte),
		SendChan:    make(chan []byte),
		Ctx:         ctx,
		CancelCtx:   cancelCtx,
	}
	agentSocket.IConnection = fakeSocket

	wg.Add(1)
	go func() {
		defer wg.Done()
		for atomic.LoadInt64(&cntRecv) != atomic.LoadInt64(&cntTotal) || fakeSocket.isConnected {
			var (
				err        error
				packetData []byte
			)
			packetData = <-fakeSocket.SendChan
			if packetData == nil && !fakeSocket.isConnected {
				return
			}
			packetData, err = packEncrypter.Decrypt(packetData)
			if err != nil {
				b.Error("Failed to decrypt data packet:", err.Error())
			}
			recvData, err := parsePacketData(packetData)
			if err != nil {
				b.Error("Failed to receive data packet:", err.Error())
			}
			if !bytes.Equal(recvData, testData) {
				b.Error("Failed compare of received data")
			}
			atomic.AddInt64(&cntRecv, 1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-quit:
				return
			case packet := <-moduleSocket.GetReceiver():
				packet.SetAck()
				switch packet.PType {
				case PTControl:
					if packet.GetControlMsg().AgentInfo.ID != agentID {
						b.Error("Failed to get Agent ID from packet proto")
					}
					if packet.GetControlMsg().MsgType == AgentConnected {
						continue
					}
					if packet.GetControlMsg().MsgType == AgentDisconnected {
						continue
					}
					b.Error("Failed to get Message Type from packet proto")
				default:
					b.Error("Failed to parse packet type")
				}
			}
		}
	}()

	if !proto.addAgent(ctx, agentSocket) {
		b.Fatal("Failed add Agent Socket object")
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			data := &Data{Data: testData}
			if err := moduleSocket.SendDataTo(ctx, agentToken, data); err != nil {
				if errors.Is(err, websocket.ErrCloseSent) {
					break
				}
				b.Fatal("Failed to send data to fake socket:", err.Error())
			}
			atomic.AddInt64(&cntTotal, 1)
			b.SetBytes(testDataLen)
		}
	})
	b.StopTimer()

	for atomic.LoadInt64(&cntRecv) != atomic.LoadInt64(&cntTotal) {
		runtime.Gosched()
	}

	if !proto.delAgent(ctx, agentSocket) {
		b.Fatal("Failed delete Agent Socket object")
	}
	if !errors.Is(fakeSocket.Close(ctx), websocket.ErrCloseSent) {
		b.Error("Fake Socket object must be closed on delete agent api")
	}

	quit <- struct{}{}
	fakeSocket.SendChan <- nil
	wg.Wait()
}

func BenchmarkStabFakeSocketConnectAndSendFromModuleToAgent(b *testing.B) {
	ctx := context.Background()
	var cntTotal, cntRecv, cntAgents, cntCon, cntDis int64
	var wg sync.WaitGroup
	ctxAS, cancelCtxAS := context.WithCancel(ctx)
	ctxMS, cancelCtxMS := context.WithCancel(ctx)
	proto, err := getVXProto()
	if err != nil {
		b.Fatal(err)
	}
	defer proto.Close(ctx)

	moduleSocket := proto.NewModule("test", groupID)
	if moduleSocket != nil {
		if !proto.AddModule(moduleSocket) {
			b.Fatal("Failed add Module Socket object")
		}
	} else {
		b.Fatal("Failed initialize Module Socket object")
	}
	defer func() {
		if !proto.DelModule(moduleSocket) {
			b.Fatal("Failed delete Module Socket object")
		}
	}()

	testData := []byte("test data message")
	testDataLen := int64(len(testData))
	ctx, cancelCtx := context.WithCancel(ctx)
	fakeSocket := &FakeSocket{
		isConnected: true,
		RecvChan:    make(chan []byte),
		SendChan:    make(chan []byte),
		Ctx:         ctx,
		CancelCtx:   cancelCtx,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for atomic.LoadInt64(&cntRecv) != atomic.LoadInt64(&cntTotal) || fakeSocket.isConnected {
			var (
				err        error
				packetData []byte
			)

			select {
			case <-ctxAS.Done():
				continue
			case packetData = <-fakeSocket.SendChan:
			}

			packetData, err = packEncrypter.Decrypt(packetData)
			if err != nil {
				b.Error("Failed to decrypt data packet:", err.Error())
			}
			packet, err := (&Packet{}).fromBytesPB(packetData)
			if err != nil {
				b.Error("Failed to receive data packet:", err.Error())
			}
			if packet.Module != "test" {
				b.Error("Failed to match module name")
			}
			if packet.PType != PTData {
				b.Error("Failed to match packet type")
			}
			if !bytes.Equal(packet.GetData().Data, testData) {
				b.Error("Failed compare of received data")
			}
			atomic.AddInt64(&cntRecv, 1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctxMS.Done():
				return
			case packet := <-moduleSocket.GetReceiver():
				packet.SetAck()
				switch packet.PType {
				case PTControl:
					if packet.GetControlMsg().AgentInfo.ID != agentID {
						b.Error("Failed to get Agent ID from packet proto")
					}
					if packet.GetControlMsg().MsgType == AgentConnected {
						atomic.AddInt64(&cntCon, 1)
						continue
					}
					if packet.GetControlMsg().MsgType == AgentDisconnected {
						atomic.AddInt64(&cntDis, 1)
						continue
					}
					b.Error("Failed to get Message Type from packet proto")
				default:
					b.Error("Failed to parse packet type")
				}
			}
		}
	}()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		agentSocket := makeAgentSocket(proto)
		agentSocket.src = randString(20)
		agentSocket.auth.resp.Stoken = &agentSocket.src
		agentToken := randString(20)
		agentSocket.auth.req.Atoken = &agentToken
		agentSocket.auth.resp.Atoken = &agentToken
		agentSocket.IConnection = fakeSocket
		if !proto.addAgent(ctx, agentSocket) {
			b.Fatal("Failed add Agent Socket object")
		}
		for pb.Next() {
			data := &Data{Data: testData}
			if err := moduleSocket.SendDataTo(ctx, agentToken, data); err != nil {
				if errors.Is(err, websocket.ErrCloseSent) || errors.Is(err, errForceClosed) {
					break
				}
				b.Fatal("Failed to send data to fake socket:", err.Error())
			}
			atomic.AddInt64(&cntTotal, 1)
			b.SetBytes(testDataLen)
		}
		if !proto.delAgent(ctx, agentSocket) {
			b.Fatal("Failed delete Agent Socket object")
		}
		atomic.AddInt64(&cntAgents, 1)
	})
	b.StopTimer()

	for atomic.LoadInt64(&cntRecv) != atomic.LoadInt64(&cntTotal) {
		runtime.Gosched()
	}
	for atomic.LoadInt64(&cntCon) != atomic.LoadInt64(&cntAgents) {
		runtime.Gosched()
	}
	for atomic.LoadInt64(&cntDis) != atomic.LoadInt64(&cntAgents) {
		runtime.Gosched()
	}

	cancelCtxAS()
	if !errors.Is(fakeSocket.Close(ctx), websocket.ErrCloseSent) {
		b.Error("Fake Socket object must be closed on delete agent api")
	}

	cancelCtxMS()
	wg.Wait()
}

func getVXProto() (*vxProto, error) {
	ivxp, err := New(&FakeMainModule{})
	if err != nil {
		return nil, fmt.Errorf("failed to create an IVXProto object: %w", err)
	}
	p, ok := ivxp.(*vxProto)
	if !ok {
		return nil, fmt.Errorf("failed to type assert the test IVXProto")
	}
	return p, nil
}
