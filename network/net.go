package network

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/cloudflare/cfssl/log"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/multiformats/go-multiaddr"
	chain2 "github.com/ssbcV2/chain"
	"github.com/ssbcV2/commonconst"
	"github.com/ssbcV2/meta"
	"github.com/ssbcV2/redis"
	"github.com/ssbcV2/util"
	"github.com/ssbcV2/vrf"
	"io"
	"sync"
	"time"
)

var mutex = &sync.Mutex{}

var Proposals []meta.ProposalSign

// MakeBasicHost creates a LMibP2P host.
func MakeBasicHost(port int) host.Host {
	// Creates a new RSA key pair for this host.
	var r io.Reader
	r = rand.Reader
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	basicHost, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)

	if err != nil {
		panic(err)
	}
	return basicHost
}

//在本机找一个可用的端口号
func AvailablePort(h host.Host) (port string) {
	for _, la := range h.Network().ListenAddresses() {
		if p, err := la.ValueForProtocol(multiaddr.P_TCP); err == nil {
			port = p
			break
		}
	}
	if port == "" {
		panic("was not able to find actual local port")
	}
	return
}

func ShowHostAddresses(host host.Host) {
	for _, la := range host.Addrs() {
		log.Infof(" - %v\n", la)
	}
}

func HandleStream(s network.Stream) {
	log.Info("Got a new stream")
	//创建一个buff stream来不阻塞读和写
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	go ReadData(rw)
	go WriteData(rw)
}

func ReadData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		log.Info("Received Msg:", str)
		if err != nil {
			log.Error(err.Error())
		}
		if str == "" {
			return
		}
		if str != "\n" {
			//对消息进行解析
			msg := ParseP2PMessage(str)
			if msg.Type == commonconst.BlockChainSynchronizeMsg {
				//区块同步消息
				handleBlockChainSynchronizeMsg(msg.Content)
			} else if msg.Type == commonconst.AbstractBlockHeaderSynchronizeMsg {
				//抽象区块头同步消息
				handleAbstractBlockHeaderSynchronizeMsg(msg.Content)
			} else if msg.Type == commonconst.VRFOrderMsg {
				//vrf指令处理
				handleVRFOrderMsg(msg.Content, rw)
			} else if msg.Type == commonconst.VRFMsg {
				//vrf结果消息处理
				handleVRFMsg(msg.Content, rw)
			} else if msg.Type == commonconst.TssMsg {
				//针对提案的门限签名的消息处理
				//节点收到提案，基于提案进行签名发给提案节点，逐一验证，达到阈值后签名成功
			} else if msg.Type == commonconst.AbstractHeaderProposalMsg {
				//收到提案节点的提案消息的处理
				handleAbstractHeaderProposalMsg(msg.Content, rw)
			} else if msg.Type == commonconst.HeaderProposalSignMsg {
				//提案节点收到针对提案的签名
				handleHeaderProposalSignMsg(msg.Content)
			}
		}
	}
}

//提案节点收到针对提案的签名处理
func handleHeaderProposalSignMsg(content string) {
	log.Info("Signature Verifying:", content)
	//首先解析出content
	var s meta.ProposalSign
	err := json.Unmarshal([]byte(content), &s)
	if err != nil {
		log.Error("[handleHeaderProposalSignMsg] json unmarshal failed,err=", err)
	}
	//对签名进行验签
	ok := util.RSAVerifySign(&s.PubKey, s.Hash, s.Sign)
	if ok {
		log.Info("Proposal Signature Verification is Successful")
		//验签成功后加入签名集
		Proposals = append(Proposals, s)
	} else {
		log.Error("Proposal Signature Verification Failed")
	}

	//判断是否达到阈值的要求
	if len(Proposals) >= s.Threshold {
		log.Info("The Proposal Signature Reaches The Threshold Requirement, Local Node Submits The Abstract Block Header for Storage! ", "Abstract Block Header:", s.Hash)
		//达到阈值要求，提交存储最终的抽象区块头
		summitAbHeaders(s.Hash)
	} else {
		log.Info("签名未达到阈值要求")
	}
}

func summitAbHeaders(headers []byte) {
	redis.SetIntoRedis(commonconst.AbstractHeadersFinalKey, string(headers))
	log.Info("The Abstract Block Header Is Stored Successfully")
}

//处理提案节点的提案消息
func handleAbstractHeaderProposalMsg(content string, rw *bufio.ReadWriter) {
	log.Info("Verifying Proposal……")
	//首先解析出content
	var abHeaders []meta.AbstractBlockHeader
	err := json.Unmarshal([]byte(content), &abHeaders)
	if err != nil {
		log.Info("[handleAbstractHeaderProposalMsg] json unmarshal failed,err=", err)
	}
	//然后获取到自己节点同步的抽象头
	localHs := getLocalStoredAbHeaders()
	//对两者进行对比验证
	abHeadersByte, _ := json.Marshal(abHeaders)
	localHsByte, _ := json.Marshal(localHs)
	h1, _ := util.CalculateHash(abHeadersByte)
	h2, _ := util.CalculateHash(localHsByte)
	if string(h1) == string(h2) {
		//说明验证成功
		log.Info("Proposal Verification Successful, Local Node Make Signs")
		//走签名流程
		//使用私钥进行签名
		sig := util.RSASign(h1, util.LocalPrivateKey)
		//将签名广播
		var s = meta.ProposalSign{
			Hash:      h1,
			PubKey:    *util.LocalPublicKey,
			Sign:      sig,
			Threshold: commonconst.ProposalSignCount,
		}
		sByte, _ := json.Marshal(s)
		var msg = meta.P2PMessage{
			Type:    commonconst.HeaderProposalSignMsg,
			Content: string(sByte),
		}
		SendP2PMessage(msg, rw)
	} else {
		log.Info("提案验证失败，拒绝签名")
	}

}

//处理抽象区块头同步消息
func handleAbstractBlockHeaderSynchronizeMsg(content string) {
	//log.Info("接收到抽象区块头")
	//首先解析出content
	//var abHeaders []meta.AbstractBlockHeader
	//err:=json.Unmarshal([]byte(content),&abHeaders)
	//if err!=nil{
	//	log.Info("[handleAbstractBlockHeaderSynchronizeMsg] json unmarshal failed,err=",err)
	//}
	//将抽象区块头信息保存至本地
	redis.SetIntoRedis(commonconst.AbstractHeadersKey, content)
}

//获取本地存储的抽象区块头列表
func getLocalStoredAbHeaders() []meta.AbstractBlockHeader {
	val, err := redis.GetFromRedis(commonconst.AbstractHeadersKey)
	if err != nil {
		log.Info(err)
	}
	var abHeaders []meta.AbstractBlockHeader
	err = json.Unmarshal([]byte(val), &abHeaders)
	if err != nil {
		log.Info("[getLocalStoredAbHeaders] json unmarshal failed,err=", err)
	}
	return abHeaders
}

func handleVRFMsg(content string, rw *bufio.ReadWriter) {
	//首先解析出content
	var vrfR meta.VRFResult
	err := json.Unmarshal([]byte(content), &vrfR)
	if err != nil {
		log.Info("[handleVRFMsg] json unmarshal failed,err=", err)
	}
	//首先对接收到的结果进行合法性验证
	log.Info("Verifying the Received Vrf Results:", vrfR)
	ok := vrf.VerifyVrf(vrfR)
	if !ok {
		log.Info("对接收到的结果验证失败")
	}
	//然后将结果保存至vrf集
	RefResultList = append(RefResultList, vrfR)
	//判断是否收集够所有的vrf进行提案节点决策
	if len(RefResultList) >= vrfR.Count {
		//收集足够
		//根据是否是最大的判定自身是否具备提案权
		if compareToJudge(RefResultList, LocalRefResult) {
			log.Info("Local Node Has Right To Propose")
			//判定自己已具备提案权-->发起提案
			//首先获取到本地存储的同步到的抽象区块头列表
			localAbHeaders := getLocalStoredAbHeaders()
			//发送p2p提案
			log.Info("Local Node Initiates An Abstract Block Header Storage Proposal")
			sendProposal(localAbHeaders, rw)
		} else {
			log.Info("Local Node Not Has Right To Propose")
		}
	} else {
		log.Info("vrf结果集未收集完全")
	}
}

//发送p2p抽象区块头提案消息
func sendProposal(abHeaders []meta.AbstractBlockHeader, rw *bufio.ReadWriter) {
	var p2pMsg meta.P2PMessage
	content, _ := json.Marshal(abHeaders)
	p2pMsg = meta.P2PMessage{
		Type:    commonconst.AbstractHeaderProposalMsg,
		Content: string(content),
	}
	SendP2PMessage(p2pMsg, rw)
}

//基于所有的vrf结果集决策自己是否具备提案权
func compareToJudge(vl []meta.VRFResult, localV meta.VRFResult) bool {
	for i, v := range vl {
		log.Info("Current Received VRF Results:", "index=", i, ",result=", v.Result)
	}
	log.Info("Local VRF Result:", localV.Result)
	var max float64
	max = 0
	for _, v := range vl {
		tempR := v.Result
		if tempR > max {
			max = tempR
		}
	}
	if max == localV.Result {
		return true
	} else {
		return false
	}
}
func handleVRFOrderMsg(content string, rw *bufio.ReadWriter) {
	//本地运行vrf函数
	//step1：每个节点本地运行可验证随机函数，将随机数结果及验证材料发送至其他节点
	vrfR := vrf.GenerateVrfResult(content)
	//先将存储集重置
	RefResultList = make([]meta.VRFResult, 0)
	//保存本地结果
	RefResultList = append(RefResultList, vrfR)
	LocalRefResult = vrfR
	//给其他节点发自己的随机结果及附带验证
	go SendVrfResult(vrfR, rw)
}

//接收到区块同步消息后的处理程序
func handleBlockChainSynchronizeMsg(content string) {
	chain := make([]meta.Block, 0)
	if err := json.Unmarshal([]byte(content), &chain); err != nil {
		log.Error(err.Error())
	}
	log.Info("Receive block chain")
	fmt.Printf("\x1b[32m%s\x1b[0m> ", content)
	mutex.Lock()
	//对比收到的区块链与本地的，进行本地更新
	chain2.UpdateChain(chain)
	mutex.Unlock()
}

func ParseP2PMessage(s string) meta.P2PMessage {
	var msg meta.P2PMessage
	//将s反序列化
	err := json.Unmarshal([]byte(s), &msg)
	if err != nil {
		log.Info("[ParseP2PMessage] json unmarshal failed,err=", err)
	}
	return msg
}

func WriteData(rw *bufio.ReadWriter) {
	//单协程进行区块协同
	//go BlockChainSynchronize(rw)
	//单协程向他链节点进行抽象区块头同步
	//go BlockChainHeaderSynchronize(rw)
	//解析终端命令
	ParseClientOrder(rw)
}

//区块链同步
func BlockChainSynchronize(rw *bufio.ReadWriter) {
	for {
		//每1分钟节点之间同步一次
		bc := chain2.GetCurrentBlockChain()
		bcByte, _ := json.Marshal(bc)
		msg := meta.P2PMessage{
			Type:    commonconst.BlockChainSynchronizeMsg, //区块链同步
			Content: string(bcByte),                       //区块链信息
		}
		SendP2PMessage(msg, rw)
		time.Sleep(60 * time.Second)
	}
}

//抽象区块头同步
func BlockChainHeaderSynchronize(rw *bufio.ReadWriter) {
	for {
		//每10秒抽象区块头节点之间同步一次
		abhs := chain2.GetLocalAbstractBlockChainHeaders("ssbc")
		bcByte, _ := json.Marshal(abhs)
		msg := meta.P2PMessage{
			Type:    commonconst.AbstractBlockHeaderSynchronizeMsg, //抽象区块头同步
			Content: string(bcByte),                                //区块链信息
		}
		SendP2PMessage(msg, rw)
		time.Sleep(10 * time.Second)
	}
}

//发送消息
func SendP2PMessage(msg meta.P2PMessage, rw *bufio.ReadWriter) {
	//首先消息进行序列化
	mutex.Lock()
	bytes, err := json.Marshal(msg)
	if err != nil {
		log.Info(err)
	}
	//然后将消息发送
	mutex.Unlock()
	mutex.Lock()
	rw.WriteString(fmt.Sprintf("%s\n", string(bytes)))
	rw.Flush()
	mutex.Unlock()
}
