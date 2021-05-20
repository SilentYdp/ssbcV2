package network

import (
	"encoding/json"
	"github.com/cloudflare/cfssl/log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/ssbcV2/chain"
	"github.com/ssbcV2/commonconst"
	"github.com/ssbcV2/merkle"
	"github.com/ssbcV2/meta"
	"github.com/ssbcV2/redis"
	"github.com/ssbcV2/util"
	"net"
	"strconv"
	"strings"
)

var TCPConnMap map[string]net.Conn

func init() {
	TCPConnMap = make(map[string]net.Conn)
}

//使用tcp发送消息
func TCPSend(msg meta.TCPMessage, addr string) {

	//版本1：复用连接
	//版本2：每次建立新的连接
	//var con net.Conn
	//先判断该连接之前是否已有
	//if exCon, exists := TCPConnMap[addr]; exists {
	//	//直接复用之前的连接
	//	con = exCon
	//} else {
		//否则直接dial建立新的连接
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Error("[TCPSend]connect error,err:", err, "msg:", msg, "addr:", addr)
		return
		//}
		//TCPConnMap[addr] = conn
		//con = conn
	}
	context, _ := json.Marshal(msg)
	_, err = conn.Write(context)
	if err != nil {
		log.Error(err)
	}

	//保持连接，可以复用，不用关闭
	//暂时先不复用，会有死循环的问题
	defer conn.Close()
}

func ServerConnHandler(c net.Conn) {
	//1.conn是否有效
	if c == nil {
		log.Error("无效的 TCP 连接")
	}

	//2.新建网络数据流存储结构
	buf := make([]byte, 4096)
	//3.循环读取网络数据流
	for {
		//3.1 网络数据流读入 buffer
		cnt, err := c.Read(buf)
		//3.2 数据读尽、读取错误 关闭 socket 连接
		if cnt == 0 || err != nil {
			//c.Close()  --感觉不需要关闭这个连接，因为是长连接
			break
		}

		//3.3 根据输入流进行逻辑处理
		//buf数据 -> 去两端空格的string
		inStr := strings.TrimSpace(string(buf[0:cnt]))
		//去除 string 内部"-"
		//解析tcp消息
		var msg meta.TCPMessage
		err = json.Unmarshal([]byte(inStr), &msg)
		if err != nil {
			log.Error("ServerConnHandler json unmarshal failed,err=", err)
		}
		//获取消息类型
		msgType := msg.Type

		//log.Info("对方节点:",c.RemoteAddr(),"传输->" + inStr)

		switch msgType {
		case commonconst.TcpPing:
			handleTcpPing(msg, c)
		case commonconst.TcpAbstractHeader:
			handleTcpAbstractHeader(msg, c)
		case commonconst.TcpCrossTrans:
			//接收到对方链的跨链交易处理
			handleTcpCrossTrans(msg, c)
		default:
			c.Write([]byte("服务器端回复" + inStr + "\n"))
		}
		//c.Close() //关闭client端的连接，telnet 被强制关闭
		//fmt.Printf("来自 %v 的连接关闭\n", c.RemoteAddr())
	}
}

//处理他链传输的跨链交易
func handleTcpCrossTrans(msg meta.TCPMessage, c net.Conn) {
	//首先解析出跨链交易
	log.Info("Local Node ", c.LocalAddr(), "Received Cross Transaction From Remote Node ", c.RemoteAddr())
	log.Info("Cross Transaction:", msg.Content)
	tr := msg.Content
	var ct meta.CrossTran
	err := json.Unmarshal([]byte(tr), &ct)
	if err != nil {
		log.Error("handleTcpCrossTrans,json ct failed", err)
	}
	switch ct.Type {
	case commonconst.CrossTranTransferType:
		//转账交易处理
		handleRemoteCrossTransfer(ct, c)
	}
}

//处理他链转来的转账交易
func handleRemoteCrossTransfer(ct meta.CrossTran, c net.Conn) {
	log.Info("Merkle Proof Verifying……")
	//首先基于proof进行merkle验证
	proof := ct.Proof
	//先获取到已同步的抽象区块头
	key := commonconst.RemoteHeadersKey + ct.SourceChainId
	abHsStr, err := redis.GetFromRedis(key)
	if err != nil {
		log.Error(err)
	}
	var abHs []meta.AbstractBlockHeader
	err = json.Unmarshal([]byte(abHsStr), &abHs)
	if err != nil {
		log.Error("handleTcpCrossTrans,json abHs failed", err)
	}
	//获取到抽象区块头的merkle root
	merkleRoot := abHs[ct.Proof.Height].MerkleRoot
	//基于转账交易的merkle proof以及同步到的抽象区块头进行merkle验证
	if merkle.VerifyTranExistence(proof.TransHash, proof.MerklePath, proof.MerkleIndex, merkleRoot) {
		//如果验证成功，那么进行本链的转账交易
		newTx := meta.Transaction{
			From:  "",
			To:    ct.To,
			Data:  meta.TransactionData{},
			Value: ct.Value,
			Id:    nil,
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
		//先不打印
		//spew.Dump(curBlockChain)
		//重置当前的交易列表
		chain.ClearCurrentTxs()
		//生成新区块后保持与他链的抽象区块头同步
		log.Info("Abstract Header Synchronizing")
		//向对方发送自己的抽象区块头列表
		LocalAHs := chain.GetLocalAbstractBlockChainHeaders(commonconst.LocalChainId2)
		//然后基于tcp长连接发送给对方服务节点
		LocalChainAbstractHByte, _ := json.Marshal(LocalAHs)
		resp := meta.TCPMessage{
			Type:    commonconst.TcpAbstractHeader, //跨链抽象区块头同步
			Content: LocalChainAbstractHByte,
		}
		respByte, _ := json.Marshal(resp)
		c.Write(respByte)
		//然后打包跨链交易通过中继节点发送至对方链
		//首先根据交易Id查询出所在的区块高度，以及
		blockHeight, sequence := chain.LocateBlockHeightWithTran(tId)
		//打包跨链交易回执
		packedTranReceipt := chain.PackCrossReceipt(ct, blockHeight, sequence)
		//将打包好的跨链交易发送至对方链上
		packedTranReceiptByte, _ := json.Marshal(packedTranReceipt)
		receMess := meta.TCPMessage{
			Type:    commonconst.TcpCrossTransReceipt,
			Content: packedTranReceiptByte,
		}
		receMessByte, _ := json.Marshal(receMess)
		c.Write(receMessByte)
		log.Info("Local Node:", c.LocalAddr(), " Send Cross Receipt To Dest Chain Node ", c.RemoteAddr())
		//log.Info("Cross Receipt:", packedTranReceipt)
	}

}

//处理客户端发过来的区块头
func handleTcpAbstractHeader(msg meta.TCPMessage, c net.Conn) {
	log.Info("Local Node ", c.LocalAddr(), "Has Received Abstract Headers From Remote Node ", c.RemoteAddr())
	log.Info("Abstract Headers:", msg.Content)
	//解析对方链的区块头
	remoteBH := msg.Content
	//将对方链的区块头存储到本地
	var rbhs []meta.AbstractBlockHeader
	err := json.Unmarshal([]byte(remoteBH), &rbhs)
	if err != nil {
		log.Info("handleTcpAbstractHeader json unmarshal failed,err=", err)
	}
	//解析出对方的chainID
	chainId := rbhs[0].ChainId
	//生成存储key
	key := commonconst.RemoteHeadersKey + chainId
	redis.SetIntoRedis(key, string(remoteBH))

	log.Info("Has saved ", chainId, "'s Abstract Headers To Local")

	//向对方发送自己的抽象区块头列表
	LocalAHs := chain.GetLocalAbstractBlockChainHeaders(commonconst.LocalChainId2)
	//然后基于tcp长连接发送给对方服务节点
	LocalChainAbstractHByte, _ := json.Marshal(LocalAHs)
	resp := meta.TCPMessage{
		Type:    commonconst.TcpAbstractHeader, //跨链抽象区块头同步
		Content: LocalChainAbstractHByte,
	}
	respByte, _ := json.Marshal(resp)
	c.Write(respByte)
}

func handleTcpPing(msg meta.TCPMessage, c net.Conn) {
	var resp meta.TCPMessage
	resp = meta.TCPMessage{
		Type:    commonconst.TcpPong,
		Content: nil,
	}
	respByte, _ := json.Marshal(resp)
	c.Write(respByte)
}

//开启serverSocket
func ServerSocket(host host.Host) {
	//获取到可用的tcp连接端口
	port := AvailablePort(host)

	portInt, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		log.Error(err)
	}

	portInt = portInt + 1

	portStr := strconv.FormatInt(portInt, 10)

	//生成监听地址
	address := "127.0.0.1:" + portStr

	//1.监听端口
	server, err := net.Listen("tcp", address)

	if err != nil {
		log.Error("开启socket服务失败")
	}

	log.Info("TCP Waiting Shaking Hands，Listening Address:", address)

	for {
		//2.接收来自 client 的连接,会阻塞
		conn, err := server.Accept()
		if err != nil {
			log.Error("连接出错")
		}

		//并发模式 接收来自客户端的连接请求，一个连接 建立一个 conn，服务器资源有可能耗尽 BIO模式
		go ServerConnHandler(conn)
	}

}

/**
  client 发送端 程序
  问题：如何区分  c net.Conn 的 Write 与 Read 的数据流向?
      1. c.Write([]byte("hello"))
         c <- "hello"
      2. c.Read(buf []byte)
         c -> buf (空buf)
  客户端 和 服务器端都有 Close conn 的功能
*/

func ClientConnHandler(c net.Conn, msg meta.TCPMessage) {
	//判断con是否已关闭，若关闭则重新连接
	msgByte, _ := json.Marshal(msg)
	log.Info("Local Node ", c.LocalAddr(), " Send Msg To Remote node ", c.LocalAddr(), " Msg:", string(msgByte))
	//去除输入两端空格
	//msg = strings.TrimSpace(msg)
	//客户端请求数据写入 conn，并传输
	c.Write(msgByte)

	//接收服务器返回数据
	//缓存 conn 中的数据
	buf := make([]byte, 4096)
	//服务器端返回的数据写入空buf
	cnt, err := c.Read(buf)

	if err != nil {
		log.Errorf("读取数据失败 %s\n", err)
	}

	//buf数据 -> 去两端空格的string
	inStr := strings.TrimSpace(string(buf[0:cnt]))
	//解析tcp消息
	var resp meta.TCPMessage
	err = json.Unmarshal([]byte(inStr), &resp)
	if err != nil {
		log.Error("ServerConnHandler json unmarshal failed,err=", err)
	}
	//获取消息类型
	respType := resp.Type

	log.Info("Local Node ", c.LocalAddr(), " Received Msg From Remote Node ", c.RemoteAddr(), " Msg:", inStr)

	switch respType {
	case commonconst.TcpPong:
		handleTcpPong(resp, c)
	case commonconst.TcpAbstractHeader:
		//接收到服务端的区块头信息的回复
		handleTcpAbstractHeaderResp(resp, c)
	case commonconst.TcpCrossTransReceipt:
		//接收到服务端的关于交易执行的回执
		handleTcpCrossTransReceipt(resp, c)
	default:
	}
}

func handleTcpCrossTransReceipt(msg meta.TCPMessage, c net.Conn) {
	log.Info("Local Node ", c.LocalAddr(), " Has Received Remote Node ", c.RemoteAddr(), " Sent Transaction Receipt")
	log.Info("Transaction Receipt:", msg.Content)
	//解析回执
	var receipt meta.CrossTranReceipt
	err := json.Unmarshal([]byte(msg.Content), &receipt)
	if err != nil {
		log.Error("handleTcpCrossTransReceipt json failed", err)
	}
	//首先基于proof进行merkle验证
	log.Info("Transaction Receipt Merkle Proof Verifying,Receipt Proof:", receipt.Proof)
	proof := receipt.Proof
	//先获取到已同步的抽象区块头
	key := commonconst.RemoteHeadersKey + receipt.SourceChainId
	abHsStr, err := redis.GetFromRedis(key)
	if err != nil {
		log.Error(err)
	}
	var abHs []meta.AbstractBlockHeader
	err = json.Unmarshal([]byte(abHsStr), &abHs)
	if err != nil {
		log.Error("handleTcpCrossTrans,json abHs failed", err)
	}
	//获取到抽象区块头的merkle root
	merkleRoot := abHs[receipt.Proof.Height].MerkleRoot
	//基于转账交易的merkle proof以及同步到的抽象区块头进行merkle验证
	if merkle.VerifyTranExistence(proof.TransHash, proof.MerklePath, proof.MerkleIndex, merkleRoot) {
		log.Info("Cross Transaction Has Finished Successfully!")
	} else {
		log.Info("Cross Transaction Has Failed!")
	}
}

func handleTcpAbstractHeaderResp(msg meta.TCPMessage, c net.Conn) {
	log.Info("Local Node ", c.LocalAddr(), " Has Received Abstract Block Headers From Remote Node ", c.RemoteAddr())
	log.Info("Abstract block headers:", msg.Content)
	//解析对方链的区块头
	remoteBH := msg.Content
	//将对方链的区块头存储到本地
	var rbhs []meta.AbstractBlockHeader
	err := json.Unmarshal([]byte(remoteBH), &rbhs)
	if err != nil {
		log.Error("handleTcpAbstractHeader json unmarshal failed,err=", err)
	}
	//解析出对方的chainID
	chainId := rbhs[0].ChainId
	//生成存储key
	key := commonconst.RemoteHeadersKey + chainId
	redis.SetIntoRedis(key, string(remoteBH))
	log.Info("Has Saved ", chainId, "'s Abstract Header To Local")
}

func handleTcpPong(msg meta.TCPMessage, c net.Conn) {
	log.Info("Local Node ", c.LocalAddr(), " Shake Hands With Remote Node ", c.RemoteAddr(), " Successfully")
}

func ClientSocket(addr string) net.Conn {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Error("TCP建立连接失败")
		return nil
	}
	return conn
}
