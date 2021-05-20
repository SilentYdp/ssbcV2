package main

import (
	"encoding/json"
	"fmt"
	"github.com/cloudflare/cfssl/log"
	"github.com/ssbcV2/chain"
	"github.com/ssbcV2/commonconst"
	"github.com/ssbcV2/levelDB"
	"github.com/ssbcV2/meta"
	"github.com/ssbcV2/network"
	"github.com/ssbcV2/util"
	"io/ioutil"
	"net"
)

//客户端使用的tcp监听
func clientTcpListen() {
	listen, err := net.Listen("tcp", clientAddr)
	if err != nil {
		log.Error(err)
	}
	defer listen.Close()
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Error(err)
		}
		fmt.Println("新连接：", conn.LocalAddr().String(), conn.RemoteAddr().String())
		clientHandleNewConn(conn)
	}
}

func clientHandleNewConn(conn net.Conn) {
	//buf := make([]byte, 4096)
	//n, err := conn.Read(buf) //从conn读取
	//fmt.Println("接收到消息：", string(buf[:n]))
	//if err == nil {
	//	clientHandleTcpMsg(buf[:n], conn)
	//} else if err != io.EOF {
	//	log.Error(err)
	//}
	//defer conn.Close()

	//Version2
	b,err:=ioutil.ReadAll(conn)
	if err!=nil{
		log.Error("[clientHandleNewConn] err:",err)
	}
	clientHandleTcpMsg(b,conn)

}

//客户端对其他节点发来的tcp消息进行处理
func clientHandleTcpMsg(content []byte, conn net.Conn) {
	//先解析消息
	msg := network.ParseTCPMsg(content)
	//根据消息类别选择handle函数
	switch msg.Type {
	case commonconst.PBFTReply:
		bcs := chain.GetCurrentBlockChain()
		newBC := new(meta.Block)
		bc := msg.Content
		err := json.Unmarshal(bc, newBC)
		util.DealJsonErr("clientHandleTcpMsg", err)
		//判断是否已将reply的新区块入库了
		if newBC.Height == len(bcs) {
			bcs = append(bcs, *newBC)
			chain.StoreBlockChain(bcs)
			//状态更新
			refreshState(*newBC)
		}
	//接收来自主节点的区块链同步回复
	case commonconst.BlockSynResMsg:
		network.HandleBlockSynResMsg(msg, conn)
	default:
		log.Error("[clientHandleTcpMsg] invalid tcp msg type:", msg.Type)
	}
}

func refreshState (b meta.Block) {
	//ste1：首先取出本区块中所有的交易
	txs:=b.TX
	//每一笔交易写集进行更新
	for _,tx:=range txs{
		set:=tx.Data.Set
		for k,v:=range set{
			if k!=""{
				levelDB.DBPut(k,[]byte(v))
			}
		}
	}
}

//客户端监听用户发起的http请求
func clientHttpListen() {
	s := NewClientServer(commonconst.ClientToUserAddr)
	s.Start()
}

//节点使用的tcp监听
func (p *pbft) tcpListen() {
	listen, err := net.Listen("tcp", p.node.addr)
	if err != nil {
		log.Error(err)
	}
	fmt.Printf("节点开启监听，地址：%s\n", p.node.addr)
	defer listen.Close()
	for {
		conn, err := listen.Accept()
		fmt.Println("新连接：", conn.LocalAddr().String(), conn.RemoteAddr().String())
		if err != nil {
			log.Error(err)
		}
		p.handleNewConn(conn)
	}
}

func (p *pbft) handleNewConn(conn net.Conn) {
	//for {
	//
	//	buf := make([]byte, 4096)
	//	n, err := conn.Read(buf) //从conn读取
	//	if err == nil {
	//		fmt.Println("接收到消息：", string(buf[:n]))
	//		p.handleRequest(buf[:n], conn)
	//	} else if err != io.EOF {
	//		log.Error("[handleNewConn] err:", err)
	//	}
	//}

	//Version2
	b,err:=ioutil.ReadAll(conn)
	if err!=nil{
		log.Error("[handleNewConn] err:",err)
	}
	p.handleRequest(b,conn)
}



//使用tcp发送消息
func tcpDial(context []byte, addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Error("connect error", err)
		return
	}

	_, err = conn.Write(context)
	if err != nil {
		log.Fatal(err)
	}
	conn.Close()
}
