package protoagent

import (
	"fmt"

	//nolint:staticcheck  // TODO: we should consider updating this import
	"google.golang.org/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"

	"soldr/pkg/errors"
)

func PackProtoMessage(msg protoreflect.ProtoMessage, msgType Message_Type) ([]byte, error) {
	initConn, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the init connection request: %w", err)
	}
	packedMsg, err := PackMessage(msgType, initConn)
	if err != nil {
		return nil, fmt.Errorf("failed to pack the protobuf message: %w", err)
	}
	return packedMsg, nil
}

func PackMessage(msgType Message_Type, payload []byte) ([]byte, error) {
	messageData, err := proto.Marshal(&Message{
		Type:    msgType.Enum(),
		Payload: payload,
	})
	if err != nil {
		return nil, fmt.Errorf("error marshal request packet: %w", err)
	}
	return messageData, nil
}

func GetProtoMessagePayload(msg []byte) ([]byte, Message_Type, error) {
	var agentMsg Message
	if err := proto.Unmarshal(msg, &agentMsg); err != nil {
		return nil, Message_UNKNOWN, fmt.Errorf("failed to unmarshal the received data into an agent message: %w", err)
	}
	return agentMsg.GetPayload(), agentMsg.GetType(), nil
}

func UnpackProtoMessagePayload(dst proto.Message, payload []byte) error {
	if err := proto.Unmarshal(payload, dst); err != nil {
		return fmt.Errorf("failed to unmarshal the received agent message: %w", err)
	}
	return nil
}

func UnpackProtoMessage(dst proto.Message, msg []byte, msgType Message_Type) error {
	payload, actualMsgType, err := GetProtoMessagePayload(msg)
	if err != nil {
		return err
	}
	if actualMsgType != msgType {
		return fmt.Errorf("%w: expected agent message type: %d, got: %d", errors.ErrUnexpectedUnpackType, msgType, actualMsgType)
	}
	if err := UnpackProtoMessagePayload(dst, payload); err != nil {
		return err
	}
	return nil
}
