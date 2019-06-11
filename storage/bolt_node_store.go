package storage

import (
	"encoding/json"
	"errors"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/p2pserver/message/types"
	"github.com/ontio/ontology/p2pserver/peer"
	bolt "go.etcd.io/bbolt"
	"map/utils"
	"sort"
	"strconv"
	"time"
)

const (
	ADDR_BUCKET       = "ADDR_BUCKET"
	DEFAULT_LAT_LON   = 1000
	NODE_DB_FILE_NAME = "addr.db"
)

var bucketName = []byte(ADDR_BUCKET)

var db *bolt.DB

func InitNodeDb() {
	var err error
	db, err = bolt.Open(NODE_DB_FILE_NAME, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal("open node db failed")
		return
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists(bucketName)
		return err
	})
	if err != nil {
		log.Error("create address bucket fail")
	}
}

func CloseNodeDb() {
	err := db.Close()
	if err != nil {
		log.Error("Close node db error")
	}
}

func getSyncAddrInfoFromPeer(peer *peer.Peer) (string, int, string, error) {
	addr := peer.GetAddr()
	ip, _, err := ParseIpPort(addr)
	if err != nil {
		return "", 0, "", err
	}
	port := int(peer.GetPort())
	syncAddr := ip + ":" + strconv.Itoa(port)
	return ip, port, syncAddr, nil
}

func TryAddNodeAfterReceiveAddrMessage(addr string, services uint64, activeTime uint64) {
	ip, port, err := ParseIpPort(addr)
	if err != nil {
		log.Error(err)
		return
	}

	key := []byte(addr)
	val, _ := json.Marshal(NodeInfo{
		Ip:             ip,
		Port:           port,
		Services:       services,
		CanConnect:     false,
		LastActiveTime: activeTime,
		Lat:            DEFAULT_LAT_LON,
		Lon:            DEFAULT_LAT_LON,
	})

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return errors.New("bucket not exist")
		}
		if b.Get(key) == nil {
			err := b.Put(key, val)
			if err != nil {
				return err
			}
			go RefreshNodeLatLon(addr)
		}
		return nil
	})
	if err != nil {
		log.Error(err)
	}
}

func AddOrUpdateNodeAfterReceiveVersionMsg(peer *peer.Peer, payload types.VersionPayload, isHttp bool) {
	ip, port, addr, err := getSyncAddrInfoFromPeer(peer)
	if err != nil {
		log.Error("get addr info from peer error " + err.Error())
		return
	}

	now := NowInMs()
	key := []byte(addr)
	val, _ := json.Marshal(NodeInfo{
		Ip:             ip,
		Port:           port,
		Services:       payload.Services,
		Height:         payload.StartHeight,
		IsConsensus:    payload.IsConsensus,
		IsHttp:         isHttp,
		SoftVersion:    payload.SoftVersion,
		HttpInfoPort:   payload.HttpInfoPort,
		ConsensusPort:  payload.ConsPort,
		LastActiveTime: now,
		Lat:            DEFAULT_LAT_LON,
		Lon:            DEFAULT_LAT_LON,
	})

	_ = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return errors.New("bucket not exist")
		}
		oldVal := b.Get(key)
		if oldVal == nil {
			err := b.Put(key, val)
			if err != nil {
				return err
			}
			go RefreshNodeLatLon(addr)
		} else {
			var oldAddrInfo NodeInfo
			err := json.Unmarshal(oldVal, &oldAddrInfo)
			if err != nil {
				return err
			}
			oldAddrInfo.Services = payload.Services
			oldAddrInfo.Height = payload.StartHeight
			oldAddrInfo.IsConsensus = payload.IsConsensus
			oldAddrInfo.SoftVersion = payload.SoftVersion
			oldAddrInfo.IsHttp = isHttp
			oldAddrInfo.HttpInfoPort = payload.HttpInfoPort
			oldAddrInfo.ConsensusPort = payload.ConsPort
			oldAddrInfo.LastActiveTime = now
			oldVal, _ = json.Marshal(oldAddrInfo)
			err = b.Put(key, oldVal)
			if err != nil {
				return err
			}
			go RefreshNodeLatLon(addr)
		}
		return nil
	})
}

func AddOrUpdateNodeAfterReceiveVersionAckMsg(remotePeer *peer.Peer) {
	ip, port, addr, err := getSyncAddrInfoFromPeer(remotePeer)
	if err != nil {
		log.Error("get addr info from peer error " + err.Error())
		return
	}

	now := NowInMs()
	key := []byte(addr)
	val, _ := json.Marshal(NodeInfo{
		Ip:             ip,
		Port:           port,
		Services:       remotePeer.GetServices(),
		Height:         remotePeer.GetHeight(),
		LastActiveTime: now,
		CanConnect:     true,
		Lat:            DEFAULT_LAT_LON,
		Lon:            DEFAULT_LAT_LON,
	})

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return errors.New("bucket not exist")
		}
		oldVal := b.Get(key)
		if oldVal == nil {
			err := b.Put(key, val)
			if err != nil {
				return err
			}
			go RefreshNodeLatLon(addr)
		} else {
			var oldAddrInfo NodeInfo
			err := json.Unmarshal(oldVal, &oldAddrInfo)
			if err != nil {
				return err
			}
			oldAddrInfo.Services = remotePeer.GetServices()
			oldAddrInfo.Height = remotePeer.GetHeight()
			oldAddrInfo.CanConnect = true
			oldAddrInfo.LastActiveTime = now
			oldVal, _ = json.Marshal(oldAddrInfo)
			err = b.Put(key, oldVal)
			if err != nil {
				return err
			}
			go RefreshNodeLatLon(addr)
		}
		return nil
	})
	if err != nil {
		log.Error(err)
	}
}

// Receive pong message
func UpdateNodeHeight(peer *peer.Peer, height uint64) {
	_, _, addr, err := getSyncAddrInfoFromPeer(peer)
	if err != nil {
		log.Error("get addr info from peer error " + err.Error())
		return
	}
	log.Info("update height ", addr, height)

	key := []byte(addr)

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		oldVal := b.Get(key)
		if oldVal != nil {
			var oldAddrInfo NodeInfo
			err := json.Unmarshal(oldVal, &oldAddrInfo)
			if err != nil {
				return err
			}
			oldAddrInfo.Height = height
			oldAddrInfo.LastActiveTime = NowInMs()
			oldAddrInfo.CanConnect = true
			oldVal, _ = json.Marshal(oldAddrInfo)
			err = b.Put(key, oldVal)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Error("Update height error", err)
	}
}

func ListAllNodes() []*NodeInfo {
	var res []*NodeInfo
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var value NodeInfo
			json.Unmarshal(v, &value)
			if value.Lon > DEFAULT_LAT_LON-1 {
				go RefreshNodeLatLon(string(k))
			}
			res = append(res, &value)
		}
		return nil
	})
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].CanConnect && !res[j].CanConnect {
			return true
		} else if !res[i].CanConnect && res[j].CanConnect {
			return false
		}
		if res[i].LastActiveTime > res[j].LastActiveTime {
			return true
		} else if res[i].LastActiveTime < res[j].LastActiveTime {
			return false
		} else {
			return res[i].Height >= res[j].Height
		}
	})
	return res
}

func RefreshNodeLatLon(addr string) {
	ip, _, err := ParseIpPort(addr)
	if err != nil {
		log.Error(err)
		return
	}
	key := []byte(addr)

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		oldVal := b.Get(key)
		if oldVal != nil {
			var oldNodeInfo NodeInfo
			err := json.Unmarshal(oldVal, &oldNodeInfo)
			if err != nil {
				return err
			}
			if oldNodeInfo.Lat > DEFAULT_LAT_LON-1 {
				latLon := utils.GetIpLocation(ip)
				if latLon != nil {
					oldNodeInfo.Lat = latLon.Lat
					oldNodeInfo.Lon = latLon.Lon
					oldNodeInfo.Country = latLon.Country
					oldVal, _ = json.Marshal(oldNodeInfo)
					err = b.Put(key, oldVal)
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Error("Refresh node location failed", err)
	}
}
