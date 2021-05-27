package network

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/cloudflare/cfssl/log"
	"github.com/davecgh/go-spew/spew"
	"github.com/ssbcV2/chain"
	"github.com/ssbcV2/commonconst"
	"github.com/ssbcV2/meta"
	"github.com/ssbcV2/redis"
	"github.com/ssbcV2/util"
	"github.com/ssbcV2/vrf"
	"net"
	"os"

	"strings"
)

var RefResultList []meta.VRFResult
var LocalRefResult meta.VRFResult
var TCPcon net.Conn
var TCPCONMap map[string]net.Conn

func init() {
	TCPCONMap = make(map[string]net.Conn)
}

//解析终端命令
func ParseClientOrder(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		orderData, err := stdReader.ReadString('\n')
		if err != nil {
			log.Error(err.Error())
		}
		orderData = strings.Replace(orderData, "\n", "", -1)
		//对指令做出解析
		orderType, orderContent := ParseOrder(orderData)
		if orderType == commonconst.TXOrder {
			//提交新交易
			handleTx(rw, orderContent)
		} else if orderType == commonconst.VrfOrder {
			//对基于vrf以及门限签名的预言机机制的区块头提交命令执行
			handleVrf(rw, orderContent)
		} else if orderType == commonconst.RegisterOrder {
			//对客户端的进行跨链注册指令的处理
			handleRegisterOrder(rw, orderContent)
		} else if orderType == commonconst.TCPConnectOrder {
			//对TCP握手命令的解析执行
			//handleTCPConnectOrder(rw, orderContent)
		} else if orderType == commonconst.CrossTransOrder {
			//对发送跨链交易命令的解析执行
			handleCrossTransOrder(rw, orderContent)
		} else if orderType == commonconst.RemoteChainHeaderSynchronizeOrder {
			//通过Tcp长连接发送本链抽象区块头命令执行
			handleRemoteChainHeaderSynchronizeOrder(rw, orderContent)
		}
	}
}

//跨链交易发送命令的执行
func handleCrossTransOrder(rw *bufio.ReadWriter, orderContent string) {
	//首先解析用户发起的交易，并对交易进行打包
	var cst meta.CrossTran
	err := json.Unmarshal([]byte(orderContent), &cst)
	if err != nil {
		log.Error("[handleCrossTransOrder],json unmarshal failed,err=", err)
	}
	//判断跨链交易类型
	switch cst.Type {
	case commonconst.CrossTranTransferType:
		//跨链转账交易处理
		handleCrossTransfer(rw, orderContent, cst)
	default:
	}

}

//跨链转账交易处理
func handleCrossTransfer(rw *bufio.ReadWriter, orderContent string, cst meta.CrossTran) {
	//先将资产转账至锁定账户
	newTx := meta.Transaction{
		From:  cst.From,
		To:    commonconst.LockedAccount,
		Data:  meta.TransactionData{},
		Value: 10,
	}
	newTxByte, _ := json.Marshal(newTx)
	tId, _ := util.CalculateHash(newTxByte)
	newTx.Id = tId
	//发送交易到交易列表，等待上链
	chain.StoreCurrentTx(newTx)
	//生成新区块上链
	//生成存储新区块
	//此时获取当前交易集
	curTXs := chain.GetCurrentTxs()
	newBlock := chain.GenerateNewBlock(curTXs)
	//获取到当前的区块链
	curBlockChain := chain.GetCurrentBlockChain()
	mutex.Lock()
	curBlockChain = append(curBlockChain, newBlock)
	//存储新的chain
	chain.StoreBlockChain(curBlockChain)
	mutex.Unlock()
	//打印新的区块链
	log.Info("New Block Generated")
	spew.Dump(curBlockChain)
	//重置当前的交易列表
	chain.ClearCurrentTxs()
	//将新区块链发给其他节点
	SendBlockChain(curBlockChain, rw)
	//生成新区块后保持与他链的抽象区块头同步
	handleRemoteChainHeaderSynchronizeOrder(rw, orderContent)
	//再根据链名解析合约，根据目标链名查询出与中继节点的地址的tcp链接
	connect := TCPCONMap[cst.DestChainId]
	//然后打包跨链交易通过中继节点发送至对方链
	//首先根据交易Id查询出所在的区块高度，以及
	blockHeight, sequence := chain.LocateBlockHeightWithTran(tId)
	//打包跨链交易
	packedTran := chain.PackACrossTransaction(cst, blockHeight, sequence)
	//将打包好的跨链交易发送至对方链上
	packedTranByte, _ := json.Marshal(packedTran)
	msg := meta.TCPMessage{
		Type:    commonconst.TcpCrossTrans,
		Content: packedTranByte,
	}
	if connect == nil {
		log.Error("handleRemoteChainHeaderSynchronizeOrder", "tcp connection has broken")
	} else {
		log.Info("Local Node:", connect.LocalAddr(), " Send Cross Tran to Dest Chain Node ", connect.RemoteAddr())
		//packedTranByte,_:=json.Marshal(packedTran)
		//log.Info("Cross Tran:",string(packedTranByte))
		ClientConnHandler(connect, msg)
	}
}

//通过Tcp长连接发送本链抽象区块头命令执行
func handleRemoteChainHeaderSynchronizeOrder(rw *bufio.ReadWriter, orderContent string) {
	//首先获取到本链的抽象区块头
	LocalChainAbstractH := chain.GetLocalAbstractBlockChainHeaders(commonconst.LocalChainId)
	//然后基于tcp长连接发送给对方服务节点
	LocalChainAbstractHByte, _ := json.Marshal(LocalChainAbstractH)
	msg := meta.TCPMessage{
		Type:    commonconst.TcpAbstractHeader, //跨链抽象区块头同步
		Content: LocalChainAbstractHByte,
	}
	if TCPcon == nil {
		log.Error("handleRemoteChainHeaderSynchronizeOrder	", "tcp连接已中断")
	} else {
		ClientConnHandler(TCPcon, msg)
	}
}

//TCP握手命令解析执行
//func handleTCPConnectOrder(rw *bufio.ReadWriter, orderContent string) {
//	//首先获取到注册信息
//	infoStr, _ := redis.GetFromRedis(commonconst.RegisterInformationKey)
//	//然后对注册信息进行解析
//	info := parseRegister(infoStr)
//	//然后获取到需要进行tcp握手通信的地址
//	if len(info.Relayers) > 0 {
//		addr := info.Relayers[0].IP + ":" + info.Relayers[0].Port
//		//进行握手连接
//		TCPcon = ClientSocket(addr)
//		//将本TCP连接进行登记
//		TCPCONMap[info.Id] = TCPcon
//		if TCPcon != nil {
//			msg := meta.TCPMessage{
//				Type:    commonconst.TcpPing,
//				Content: nil,
//			}
//			ClientConnHandler(TCPcon, msg)
//		}
//	}
//}

//跨链信息注册handler
func handleRegisterOrder(rw *bufio.ReadWriter, orderContent string) {
	//将跨链信息存储
	redis.SetIntoRedis(commonconst.RegisterInformationKey, orderContent)
	//在终端显示
	log.Info("Cross-chain Information Registration Successful:", orderContent)
}

//注册信息解析
func parseRegister(rs string) (rt meta.RegisterInformation) {
	rs = strings.Replace(rs, "\n", "", -1)
	var t meta.RegisterInformation
	err := json.Unmarshal([]byte(rs), &t)
	if err != nil {
		log.Error("ParseTransaction:json unmarshal failed", err)
	}
	return t
}

//交易handler
func handleTx(rw *bufio.ReadWriter, orderContent string) {
	//提交了一笔新交易
	newTX := ParseTransaction(orderContent)
	//此时获取当前交易集
	curTXs := chain.GetCurrentTxs()
	if len(curTXs) == commonconst.TxsThreshold-1 {
		//交易数满足打包区块的要求
		curTXs = append(curTXs, newTX)
		//生成存储新区块
		newBlock := chain.GenerateNewBlock(curTXs)
		//获取到当前的区块链
		curBlockChain := chain.GetCurrentBlockChain()
		mutex.Lock()
		curBlockChain = append(curBlockChain, newBlock)
		//存储新的chain
		chain.StoreBlockChain(curBlockChain)
		mutex.Unlock()
		//打印新的区块链
		//log.Info("新区块生成")
		spew.Dump(curBlockChain)
		//重置当前的交易列表
		chain.ClearCurrentTxs()
		//将新区块链发给其他节点
		SendBlockChain(curBlockChain, rw)
	} else {
		//交易存至交易列表
		chain.StoreCurrentTx(newTX)
	}
}

//vrf指令handler
func handleVrf(rw *bufio.ReadWriter, orderContent string) {
	//给其他节点同步需要进行vrf的指令
	go SendVrfOrder(orderContent, rw)
	//开始基于vrf以及门限签名实现预言机，进行抽象区块头同步（每5秒进行一次）
	//step0：每个节点与他链节点进行区块同步，接收到对方链的抽象区块头
	//step1：每个节点本地运行可验证随机函数，将随机数结果及验证材料发送至其他节点
	vrfR := vrf.GenerateVrfResult(orderContent)
	//先将存储集重置
	RefResultList = make([]meta.VRFResult, 0)
	//保存本地结果
	RefResultList = append(RefResultList, vrfR)
	LocalRefResult = vrfR
	//给其他节点发自己的随机结果及附带验证
	go SendVrfResult(vrfR, rw)

}

func ParseOrder(order string) (orderType string, orderContent string) {
	//依据:来划分指令
	s := strings.Split(order, "-")
	if len(s) == 0 {
		log.Info("invalid order")
	}
	orderType = s[0]
	if len(s) == 2 {
		orderContent = s[1]
	}
	return
}
func ParseTransaction(tx string) meta.Transaction {
	tx = strings.Replace(tx, "\n", "", -1)
	var t meta.Transaction
	t = meta.Transaction{}
	err := json.Unmarshal([]byte(tx), &t)
	if err != nil {
		log.Error("ParseTransaction:json unmarshal failed", err)
	}
	Id, err := util.CalculateHash([]byte(tx))
	if err != nil {
		log.Error(err)
	}
	t.Id = Id
	return t
}

//发送需要执行vrf的指令
func SendVrfOrder(msg string, rw *bufio.ReadWriter) {
	m := meta.P2PMessage{
		Type:    commonconst.VRFOrderMsg,
		Content: msg,
	}
	SendP2PMessage(m, rw)
}

//发送vrf结果
func SendVrfResult(v meta.VRFResult, rw *bufio.ReadWriter) {
	//首先将result进行序列化
	vByte, _ := json.Marshal(v)
	vStr := string(vByte)

	m := meta.P2PMessage{
		Type:    commonconst.VRFMsg,
		Content: vStr,
	}
	SendP2PMessage(m, rw)
}

//将本地的区块链发送给其他节点
func SendBlockChain(bc []meta.Block, rw *bufio.ReadWriter) {
	bcByte, _ := json.Marshal(bc)
	var m meta.P2PMessage
	m = meta.P2PMessage{
		Type:    commonconst.BlockChainSynchronizeMsg,
		Content: string(bcByte),
	}
	SendP2PMessage(m, rw)
}
