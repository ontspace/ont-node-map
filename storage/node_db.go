package storage

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

type NodeInfo struct {
	Ip             string  `json:"ip"`
	Port           int     `json:"port"`
	Services       uint64  `json:"services"`
	Height         uint64  `json:"height"`
	IsConsensus    bool    `json:"is_consensus"`
	SoftVersion    string  `json:"soft_version"`
	IsHttp         bool    `json:"is_http"`
	HttpInfoPort   uint16  `json:"http_info_port"`
	ConsensusPort  uint16  `json:"consensus_port"`
	LastActiveTime uint64  `json:"last_active_time"`
	CanConnect     bool    `json:"can_connect"`
	Lat            float32 `json:"lat"`
	Lon            float32 `json:"lon"`
	Country        string  `json:"country"`
}

func ParseIpPort(addr string) (string, int, error) {
	i := strings.Index(addr, ":")
	if i < 0 {
		return "", 0, errors.New("format error, " + addr)
	}
	port, err := strconv.Atoi(addr[i+1:])
	if err != nil {
		return "", 0, errors.New("cannot parse port to number, " + addr)
	}
	if port <= 0 || port >= 65535 {
		return "", 0, errors.New("[p2p]port out of bound")
	}
	return addr[:i], port, nil
}

func NowInMs() uint64 {
	return uint64(time.Now().UnixNano() / int64(time.Millisecond))
}
