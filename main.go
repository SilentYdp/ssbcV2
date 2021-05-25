package main

import (
	"github.com/cloudflare/cfssl/log"
	"github.com/ssbcV2/chain"
	"github.com/ssbcV2/commonconst"
	"github.com/ssbcV2/levelDB"
	"github.com/ssbcV2/meta"
	"github.com/ssbcV2/network"
	"github.com/ssbcV2/pbft"
	"os"
)

const nodeCount = 4

//客户端与节点通信的监听地址
var clientAddr = "127.0.0.1:8888"

//客户端与用户通信监听地址
var clientHttpAddr = ":9999"

//节点池，主要用来存储监听地址
var nodeTable map[string]string

func main() {
	//为四个节点生成公私钥
	pbft.GenRsaKeys()
	nodeTable = map[string]string{
		"N0":     "127.0.0.1:8000",
		"N1":     "127.0.0.1:8001",
		"N2":     "127.0.0.1:8002",
		"N3":     "127.0.0.1:8003",
	}
	if len(os.Args) != 2 {
		log.Error("输入的参数有误！")
	}
	nodeID := os.Args[1]
	//数据库连接
	levelDB.InitDB(nodeID)
	if nodeID == "client" {
		pbft.ClientSendMessageAndListen() //启动客户端程序
		//初始化
		initBlockChain(nodeID)
	} else if addr, ok := nodeTable[nodeID]; ok {
		p := pbft.NewPBFT(nodeID, addr)
		go p.TcpListen() //启动节点
		//初始化
		initBlockChain(nodeID)
	} else {
		log.Fatal("无此节点编号！")
	}
	select {}
}

//初始化
func initBlockChain(ID string) {
	chain.BlockChain = make([]meta.Block, 0)
	//先判断节点是否为主节点
	if ID == "N0" {
		//判断是否无创世区块
		bc := chain.GetCurrentBlockChain()
		if len(bc) == 0 {
			gb := chain.GenerateGenesisBlock()
			chain.BlockChain = append(chain.BlockChain, gb)
			chain.StoreBlockChain(chain.BlockChain)
		} else {
			chain.BlockChain = bc
		}
	} else {
		//非主节点初始化时需要和主节点进行区块信息同步
		//先生成区块链同步请求消息，再发送
		msg := network.GenBlockSynReqMsg(ID)
		log.Info("发送区块链同步消息,msg:", msg, "addr:", commonconst.NodeTable["N0"])
		network.TCPSend(msg, commonconst.NodeTable["N0"])
	}
}
