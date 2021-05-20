package network

import (
	"encoding/json"
	"github.com/ssbcV2/chain"
	"github.com/ssbcV2/commonconst"
	"github.com/ssbcV2/meta"
)

//生成区块同步请求消息
func GenBlockSynReqMsg(nodeId string) meta.TCPMessage {
	msg := meta.TCPMessage{
		Type:    commonconst.BlockSynReqMsg,
		Content: nil,
		From:    nodeId,
	}
	return msg
}

//生成区块同步回应消息
func GenBlockSynResMsg() meta.TCPMessage {
	//先获取到本节点的区块链
	bc := chain.GetCurrentBlockChain()
	//log.Info("当前区块链:", bc)
	bcByte, _ := json.Marshal(bc)
	msg := meta.TCPMessage{
		Type:    commonconst.BlockSynResMsg,
		Content: bcByte,
	}
	return msg
}
