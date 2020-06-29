/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package protocols

import (
	"fmt"

	"map/p2pserver/protocols/discovery"
	"map/p2pserver/protocols/heatbeat"
	"map/p2pserver/protocols/recent_peers"

	"github.com/hashicorp/golang-lru"
	"github.com/ontio/ontology/account"
	"github.com/ontio/ontology/common/config"
	"github.com/ontio/ontology/common/log"
	msgCommon "github.com/ontio/ontology/p2pserver/common"
	msgTypes "github.com/ontio/ontology/p2pserver/message/types"
	"github.com/ontio/ontology/p2pserver/net/protocol"
	"github.com/ontio/ontology/p2pserver/protocols/bootstrap"
	"github.com/ontio/ontology/p2pserver/protocols/reconnect"
	"github.com/ontio/ontology/p2pserver/protocols/subnet"
	"github.com/ontio/ontology/p2pserver/protocols/utils"
)

//respCache cache for some response data
var respCache, _ = lru.NewARC(msgCommon.MAX_RESP_CACHE_SIZE)

//Store txHash, using for rejecting duplicate tx
// thread safe
var txCache, _ = lru.NewARC(msgCommon.MAX_TX_CACHE_SIZE)

type MsgHandler struct {
	seeds                    *utils.HostsResolver
	reconnect                *reconnect.ReconnectService
	discovery                *discovery.Discovery
	heatBeat                 *heatbeat.HeartBeat
	bootstrap                *bootstrap.BootstrapService
	persistRecentPeerService *recent_peers.PersistRecentPeerService
	subnet                   *subnet.SubNet
	acct                     *account.Account // nil if conenesus is not enabled
	staticReserveFilter      p2p.AddressFilter
}

func NewMsgHandler(acct *account.Account, staticReserveFilter p2p.AddressFilter, logger msgCommon.Logger) *MsgHandler {
	gov := utils.NewGovNodeMockResolver(nil) //utils.NewGovNodeResolver(ld)
	seedsList := config.DefConfig.Genesis.SeedList
	seeds, invalid := utils.NewHostsResolver(seedsList)
	if invalid != nil {
		panic(fmt.Errorf("invalid seed list； %v", invalid))
	}
	subNet := subnet.NewSubNet(acct, seeds, gov, logger)
	return &MsgHandler{seeds: seeds, subnet: subNet, acct: acct, staticReserveFilter: staticReserveFilter}
}

func (self *MsgHandler) GetReservedAddrFilter(staticFilterEnabled bool) p2p.AddressFilter {
	return self.subnet.GetReservedAddrFilter(staticFilterEnabled)
}

func (self *MsgHandler) GetMaskAddrFilter() p2p.AddressFilter {
	return self.subnet.GetMaskAddrFilter()
}

func (self *MsgHandler) GetSubnetMembersInfo() []msgCommon.SubnetMemberInfo {
	return self.subnet.GetMembersInfo()
}

func (self *MsgHandler) start(net p2p.P2P) {
	self.reconnect = reconnect.NewReconectService(net, self.staticReserveFilter)
	maskFilter := self.subnet.GetMaskAddrFilter()
	self.discovery = discovery.NewDiscovery(net, config.DefConfig.P2PNode.ReservedCfg.MaskPeers, maskFilter, 0)
	self.bootstrap = bootstrap.NewBootstrapService(net, self.seeds)
	self.heatBeat = heatbeat.NewHeartBeat(net)
	self.persistRecentPeerService = recent_peers.NewPersistRecentPeerService(net)
	go self.persistRecentPeerService.Start()
	go self.reconnect.Start()
	go self.discovery.Start()
	go self.heatBeat.Start()
	go self.bootstrap.Start()
	go self.subnet.Start(net)
}

func (self *MsgHandler) stop() {
	self.reconnect.Stop()
	self.discovery.Stop()
	self.persistRecentPeerService.Stop()
	self.heatBeat.Stop()
	self.bootstrap.Stop()
	self.subnet.Stop()
}

func (self *MsgHandler) HandleSystemMessage(net p2p.P2P, msg p2p.SystemMessage) {
	switch m := msg.(type) {
	case p2p.NetworkStart:
		self.start(net)
	case p2p.PeerConnected:
		self.reconnect.OnAddPeer(m.Info)
		self.discovery.OnAddPeer(m.Info)
		self.bootstrap.OnAddPeer(m.Info)
		self.persistRecentPeerService.AddNodeAddr(m.Info.RemoteListenAddress())
		self.subnet.OnAddPeer(net, m.Info)
	case p2p.PeerDisConnected:
		self.reconnect.OnDelPeer(m.Info)
		self.discovery.OnDelPeer(m.Info)
		self.bootstrap.OnDelPeer(m.Info)
		self.subnet.OnDelPeer(m.Info)
		self.persistRecentPeerService.DelNodeAddr(m.Info.RemoteListenAddress())
	case p2p.NetworkStop:
		self.stop()
	case p2p.HostAddrDetected:
		self.subnet.OnHostAddrDetected(m.ListenAddr)
	}
}

func (self *MsgHandler) HandlePeerMessage(ctx *p2p.Context, msg msgTypes.Message) {
	log.Trace("[p2p]receive message", ctx.Sender().GetAddr(), ctx.Sender().GetID())
	switch m := msg.(type) {
	case *msgTypes.AddrReq:
		self.discovery.AddrReqHandle(ctx)
	case *msgTypes.Addr:
		self.discovery.AddrHandle(ctx, m)
	case *msgTypes.FindNodeResp:
		self.discovery.FindNodeResponseHandle(ctx, m)
	case *msgTypes.FindNodeReq:
		self.discovery.FindNodeHandle(ctx, m)

	case *msgTypes.Ping:
		self.heatBeat.PingHandle(ctx, m)
	case *msgTypes.Pong:
		self.heatBeat.PongHandle(ctx, m)

	case *msgTypes.SubnetMembersRequest:
		self.subnet.OnMembersRequest(ctx, m)
	case *msgTypes.SubnetMembers:
		self.subnet.OnMembersResponse(ctx, m)

	case *msgTypes.NotFound:
		log.Debug("[p2p]receive notFound message, hash is ", m.Hash)
	default:
		msgType := msg.CmdType()
		if msgType == msgCommon.VERACK_TYPE || msgType == msgCommon.VERSION_TYPE {
			log.Infof("receive message: %s from peer %s", msgType, ctx.Sender().GetAddr())
		}
	}
}

func (mh *MsgHandler) ReconnectService() *reconnect.ReconnectService {
	return mh.reconnect
}
