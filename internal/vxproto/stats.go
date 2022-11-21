package vxproto

import (
	"encoding/json"
	"sync/atomic"
)

type metricType int

const (
	recvNumPackets metricType = iota
	sendNumPackets
	recvNetBytes
	sendNetBytes
	recvPayloadBytes
	sendPayloadBytes
)

type ProtoStats struct {
	RecvNumPackets   int64 `json:"proto_recv_num_packets"`
	SendNumPackets   int64 `json:"proto_send_num_packets"`
	RecvNetBytes     int64 `json:"proto_recv_net_bytes"`
	SendNetBytes     int64 `json:"proto_send_net_bytes"`
	RecvPayloadBytes int64 `json:"proto_recv_payload_bytes"`
	SendPayloadBytes int64 `json:"proto_send_payload_bytes"`
}

func (s *ProtoStats) incStats(m metricType, val int64) {
	switch m {
	case recvNumPackets:
		atomic.AddInt64(&s.RecvNumPackets, val)
	case sendNumPackets:
		atomic.AddInt64(&s.SendNumPackets, val)
	case recvNetBytes:
		atomic.AddInt64(&s.RecvNetBytes, val)
	case sendNetBytes:
		atomic.AddInt64(&s.SendNetBytes, val)
	case recvPayloadBytes:
		atomic.AddInt64(&s.RecvPayloadBytes, val)
	case sendPayloadBytes:
		atomic.AddInt64(&s.SendPayloadBytes, val)
	}
}

func (s *ProtoStats) DumpStats() (map[string]float64, error) {
	var (
		statsMap    = make(map[string]float64)
		statsStruct ProtoStats
	)
	if s != nil {
		statsStruct = *s
	}
	statsData, err := json.Marshal(&statsStruct)
	if err != nil {
		return statsMap, err
	}
	if err = json.Unmarshal(statsData, &statsMap); err != nil {
		return statsMap, err
	}
	return statsMap, nil
}
