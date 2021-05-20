package network

import (
	"encoding/json"
	"github.com/cloudflare/cfssl/log"
	"github.com/ssbcV2/chain"
	"github.com/ssbcV2/commonconst"
	"github.com/ssbcV2/meta"
	"github.com/ssbcV2/util"
	"net"
)

//解析消息
func ParseTCPMsg(data []byte) meta.TCPMessage {
	msg := meta.TCPMessage{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		util.DealJsonErr("ParseTCPMsg", err)
	}
	return msg
}

//处理区块链同步请求的消息
func HandleBlockSynReqMsg(msg meta.TCPMessage, conn net.Conn) {
	//先获取到请求方的地址
	reqNode := msg.From
	reqAddr := commonconst.NodeTable[reqNode]
	//生成区块同步回应消息
	resMsg := GenBlockSynResMsg()
	log.Info("区块头同步回应消息!")
	//回复
	TCPSend(resMsg, reqAddr)
}

//处理区块链同步回复消息
func HandleBlockSynResMsg(msg meta.TCPMessage, conn net.Conn) {
	bc := make([]meta.Block, 0)
	err := json.Unmarshal(msg.Content, &bc)
	util.DealJsonErr("HandleBlockSynResMsg", err)
	chain.StoreBlockChain(bc)
}
