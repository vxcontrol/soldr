package vxproto

import (
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
	RecvNumPackets   atomic.Int64
	SendNumPackets   atomic.Int64
	RecvNetBytes     atomic.Int64
	SendNetBytes     atomic.Int64
	RecvPayloadBytes atomic.Int64
	SendPayloadBytes atomic.Int64
}

func (s *ProtoStats) incStats(m metricType, val int64) {
	switch m {
	case recvNumPackets:
		s.RecvNumPackets.Add(val)
	case sendNumPackets:
		s.SendNumPackets.Add(val)
	case recvNetBytes:
		s.RecvNetBytes.Add(val)
	case sendNetBytes:
		s.SendNetBytes.Add(val)
	case recvPayloadBytes:
		s.RecvPayloadBytes.Add(val)
	case sendPayloadBytes:
		s.SendPayloadBytes.Add(val)
	}
}

func (s *ProtoStats) DumpStats() (map[string]float64, error) {
	statsMap := make(map[string]float64, 6)
	statsMap["proto_recv_num_packets"] = float64(s.RecvNumPackets.Load())
	statsMap["proto_send_num_packets"] = float64(s.SendNumPackets.Load())
	statsMap["proto_recv_net_bytes"] = float64(s.RecvNetBytes.Load())
	statsMap["proto_send_net_bytes"] = float64(s.SendNetBytes.Load())
	statsMap["proto_recv_payload_bytes"] = float64(s.RecvPayloadBytes.Load())
	statsMap["proto_send_payload_bytes"] = float64(s.SendPayloadBytes.Load())
	return statsMap, nil
}
