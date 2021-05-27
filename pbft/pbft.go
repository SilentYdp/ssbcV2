package pbft

import (
	"encoding/hex"
	"encoding/json"
	"github.com/cloudflare/cfssl/log"
	"github.com/ssbcV2/chain"
	"github.com/ssbcV2/commonconst"
	"github.com/ssbcV2/levelDB"
	"github.com/ssbcV2/meta"
	"github.com/ssbcV2/network"
	"github.com/ssbcV2/smart_contract"
	"github.com/ssbcV2/util"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

//本地消息池（模拟持久化层），只有确认提交成功后才会存入此池
var localMessagePool = []Message{}

type node struct {
	//节点ID
	nodeID string
	//节点监听地址
	addr string
	//RSA私钥
	rsaPrivKey []byte
	//RSA公钥
	rsaPubKey []byte
}

type pbft struct {
	//节点信息
	node node
	//每笔请求自增序号
	sequenceID int
	//锁
	lock sync.Mutex
	//节点的交易池
	transPool map[int][]meta.Transaction
	//临时消息池，消息摘要对应消息本体
	messagePool map[string]Request
	//存放收到的prepare数量(至少需要收到并确认2f个)，根据摘要来对应
	prePareConfirmCount map[string]map[string]bool
	//存放收到的commit数量（至少需要收到并确认2f+1个），根据摘要来对应
	commitConfirmCount map[string]map[string]bool
	//该笔消息是否已进行Commit广播
	isCommitBordcast map[string]bool
	//该笔消息是否已对客户端进行Reply
	isReply map[string]bool
}

func NewPBFT(nodeID, addr string) *pbft {
	p := new(pbft)
	p.node.nodeID = nodeID
	p.node.addr = addr
	p.node.rsaPrivKey = p.getPivKey(nodeID) //从生成的私钥文件处读取
	p.node.rsaPubKey = p.getPubKey(nodeID)  //从生成的私钥文件处读取
	p.sequenceID = 0
	p.transPool = make(map[int][]meta.Transaction)
	p.messagePool = make(map[string]Request)
	p.prePareConfirmCount = make(map[string]map[string]bool)
	p.commitConfirmCount = make(map[string]map[string]bool)
	p.isCommitBordcast = make(map[string]bool)
	p.isReply = make(map[string]bool)
	return p
}

//处理tcp请求
func (p *pbft) handleRequest(data []byte, conn net.Conn) {
	//先解析消息
	msg := network.ParseTCPMsg(data)
	//根据消息类别选择handle函数
	if strings.HasPrefix(msg.Type, commonconst.PBFT) {
		p.handlePBFTMsg(msg)
	}
	//主节点会接收到其他节点的区块链同步消息
	if msg.Type == commonconst.BlockSynReqMsg {
		network.HandleBlockSynReqMsg(msg, conn)
	}
	//其他节点会接收到主节点的区块链同步回复
	if msg.Type == commonconst.BlockSynResMsg {
		network.HandleBlockSynResMsg(msg, conn)
	}
}

func (p *pbft) handlePBFTMsg(msg meta.TCPMessage) {
	//根据消息命令调用不同的功能
	switch msg.Type {
	case commonconst.PBFTRequest:
		p.handleClientRequest(msg.Content)
	case commonconst.PBFTPrePrepare:
		p.handlePrePrepare(msg.Content)
	case commonconst.PBFTPrepare:
		p.handlePrepare(msg.Content)
	case commonconst.PBFTCommit:
		p.handleCommit(msg.Content)
	}
}

//处理客户端发来的请求
func (p *pbft) handleClientRequest(content []byte) {
	log.Info("主节点已接收到客户端发来的request ...")
	//Step1:使用json解析出Request结构体
	r := new(Request)
	err := json.Unmarshal(content, r)
	if err != nil {
		log.Error(err)
	}

	//Step2：主节点需要先将交易存储至临时的交易池，待交易池满，打包为区块进行PBFT共识
	transMsg := r.Content
	trans := meta.Transaction{}
	err = json.Unmarshal([]byte(transMsg), &trans)
	util.DealJsonErr("handleClientRequest", err)
	//step3：主节点对交易进行验签，验签不通过的丢弃
	if !RsaVerySignWithSha256(trans.Hash,trans.Sign,[]byte(trans.PublicKey)){
		log.Error("[handleClientRequest] 验签失败!!")
		return
	}
	//*******************************************************************

	//待注释内容--测试专用(主节点作恶：篡改交易内容，hash值变化，但是无法篡改客户端签名)
	//trans.Value=100
	//tB,_:=json.Marshal(trans)
	//tH,_:=util.CalculateHash(tB)
	//trans.Hash=tH

	//*******************************************************************


	//解析交易、执行交易步骤根据交易的input生成output
	trans=p.parseAndDealTransaction(trans)
	trans.Timestamp = time.Now().String()
	trans.Id, _ = util.CalculateHash([]byte(trans.Timestamp))
	bc := chain.GetCurrentBlockChain()
	index := len(bc)
	p.transPool[index] = append(p.transPool[index], trans)
	//满足交易数则打包新区块
	if len(p.transPool[index]) == commonconst.TxsThreshold {
		//主节点接收到的交易已经到达阈值，打包新区块进行PBFT共识
		newBlock := chain.GenerateNewBlock(p.transPool[index])
		//主节点对打包区块进行签名
		blockSign:=p.RsaSignWithSha256(newBlock.Hash, p.node.rsaPrivKey)
		newBlock.Signature=blockSign
		newBlockMsg, err := json.Marshal(newBlock)
		util.DealJsonErr("handleClientRequest", err)
		r.Content = string(newBlockMsg)
		//添加信息序号
		p.sequenceIDAdd()
		//获取消息摘要
		digest := getDigest(*r)
		log.Info("已将request存入临时消息池")
		//存入临时消息池
		p.messagePool[digest] = *r
		//主节点对消息摘要进行签名
		digestByte, _ := hex.DecodeString(digest)
		signInfo := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
		//拼接成PrePrepare，准备发往follower节点
		pp := PrePrepare{*r, digest, p.sequenceID, signInfo}
		b, err := json.Marshal(pp)
		if err != nil {
			log.Error(err)
		}
		log.Info("正在向其他节点进行进行PrePrepare广播 ...")
		//进行PrePrepare广播
		msg := meta.TCPMessage{
			Type:    commonconst.PBFTPrePrepare,
			Content: b,
		}
		p.broadcast(msg)
		log.Info("PrePrepare广播完成")
	} else {
		log.Info("主节点已将交易存储至交易池，交易详情：", transMsg)
	}
}

//主节点收到交易后需要对交易进行解析处理
func (p *pbft) parseAndDealTransaction(t meta.Transaction)meta.Transaction {
	log.Info("主节点接收到的交易为:",t)
	//首先判断该笔交易是否为智能合约调用
	if t.Contract!=""{
		//先判断是合约调用还是合约部署
		if t.To==commonconst.ContractDeployAddress{
			//合约部署处理
			//step1：生成合约账户
			priKey,PubKey:= GetKeyPair()
			//将公钥进行hash
			pubHash,_:=util.CalculateHash(PubKey)
			//将公钥的前20位作为账户地址
			addr:=hex.EncodeToString(pubHash[:20])
			ad:=meta.AccountData{
				Code:         t.Data.Code,
				ContractName: t.Contract,
			}
			account:=meta.Account{
				Address:    addr,
				Balance:    0,
				Data:       ad,
				PublicKey:  PubKey,
				PrivateKey: priKey,
			}
			accountB,_:=json.Marshal(account)
			set:=make(map[string]string)
			accountKey:=commonconst.AccountPrefixKey+addr
			set[accountKey]=string(accountB)
			t.Data.Set=set
		}else {
			//合约调用处理--调用智能合约产生读写集
			err,res:=smart_contract.CallContract(t.Contract,t.Method,t.Args)
			if err!=nil{
				log.Error("合约调用失败")
				//调用失败
			}else {
				//交易的data字段赋值
				t.Data.Read=res.Read
				t.Data.Set=res.Set
			}
		}
	}else {
		//非智能合约调用交易-->即简单的转账交易(而且是简单的本链转账交易)
		t=p.dealLocalTransFer(t)
	}
	return t
}

func (p *pbft) dealLocalTransFer (t meta.Transaction) meta.Transaction{
	from:=t.From
	to:=t.To
	value:=t.Value
	//step1：简单的余额校验，转出账户是否具有转账条件
	fromKey:=commonconst.AccountPrefixKey+from
	fromA:=levelDB.DBGet(fromKey)
	fromAccount:=meta.Account{}
	err:=json.Unmarshal(fromA,&fromAccount)
	if err!=nil{
		log.Error("[dealLocalTransFer] json unmarshal failed,err:",err)
	}
	if fromAccount.Balance<value{
		return t
	}else {
		//余额够，需要进行状态变更
		toKey:=commonconst.AccountPrefixKey+to
		toA:=levelDB.DBGet(toKey)
		toAccount:=meta.Account{}
		err:=json.Unmarshal(toA,&toAccount)
		if err!=nil{
			log.Error("[dealLocalTransFer] json unmarshal failed,err:",err)
		}
		//from的钱减，to的钱加
		fromAccount.Balance=fromAccount.Balance-value
		toAccount.Balance=toAccount.Balance+value
		setMap:=make(map[string]string)
		fromRefresh,_:=json.Marshal(fromAccount)
		toRefresh,_:=json.Marshal(toAccount)
		setMap[fromKey]=string(fromRefresh)
		setMap[toKey]=string(toRefresh)
		//交易的写集赋值
		t.Data.Set=setMap
		return t
	}
}

//处理预准备消息
func (p *pbft) handlePrePrepare(content []byte) {
	log.Info("本节点已接收到主节点发来的PrePrepare ...")
	//	//使用json解析出PrePrepare结构体
	pp := new(PrePrepare)
	err := json.Unmarshal(content, pp)
	if err != nil {
		log.Error(err)
	}
	//获取主节点的公钥，用于数字签名验证
	primaryNodePubKey := p.getPubKey("N0")
	digestByte, _ := hex.DecodeString(pp.Digest)
	//首先检查所有的交易客户端签名（防止主节点作恶）
	//step1先获取到全部的交易
	block:=meta.Block{}
	err=json.Unmarshal([]byte(pp.RequestMessage.Content),&block)
	if err!=nil{
		log.Error("[handlePrePrepare] json err:",err)
	}
	for _,tx:=range block.TX{
		//验签,只要有一笔交易的验签不通过则拒绝进行prepare广播
		if !RsaVerySignWithSha256(tx.Hash,tx.Sign,[]byte(tx.PublicKey)){
			log.Info("交易签名验证失败，怀疑主节点篡改交易信息，拒绝进行prepare广播")
			return
		}
	}
	if digest := getDigest(pp.RequestMessage); digest != pp.Digest {
		log.Info("信息摘要对不上，拒绝进行prepare广播")
	} else if p.sequenceID+1 != pp.SequenceID {
		log.Info("消息序号对不上，拒绝进行prepare广播")
	} else if !p.RsaVerySignWithSha256(digestByte, pp.Sign, primaryNodePubKey) {
		log.Info("主节点签名验证失败！,拒绝进行prepare广播")
	} else {
		//序号赋值
		p.sequenceID = pp.SequenceID
		//将信息存入临时消息池
		log.Info("已将消息存入临时节点池")
		p.messagePool[pp.Digest] = pp.RequestMessage
		//节点使用私钥对其签名
		sign := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
		//拼接成Prepare
		pre := Prepare{pp.Digest, pp.SequenceID, p.node.nodeID, sign}

		//*******************************************************************

		//待注释--测试专用（从节点作恶：篡改消息摘要）
		//if p.node.nodeID=="N1"{
		//	log.Info("从节点N1作恶：篡改消息摘要")
		//	pre.Digest="就是玩儿"
		//}
		//
		//if p.node.nodeID=="N2"{
		//	log.Info("从节点N2作恶：篡改消息摘要")
		//	pre.Digest="就是玩儿"
		//}

		//*******************************************************************
		bPre, err := json.Marshal(pre)
		if err != nil {
			log.Error(err)
		}
		//进行准备阶段的广播
		log.Info("正在进行Prepare广播 ...")
		msg := meta.TCPMessage{
			Type:    commonconst.PBFTPrepare,
			Content: bPre,
		}
		p.broadcast(msg)
		log.Info("Prepare广播完成")
	}
}

//处理准备消息
func (p *pbft) handlePrepare(content []byte) {
	//使用json解析出Prepare结构体
	pre := new(Prepare)
	err := json.Unmarshal(content, pre)
	if err != nil {
		log.Error(err)
	}
	log.Infof("本节点已接收到%s节点发来的Prepare ... \n", pre.NodeID)
	//获取消息源节点的公钥，用于数字签名验证
	MessageNodePubKey := p.getPubKey(pre.NodeID)
	digestByte, _ := hex.DecodeString(pre.Digest)
	if _, ok := p.messagePool[pre.Digest]; !ok {
		log.Info("当前临时消息池无此摘要，拒绝执行commit广播")
	} else if p.sequenceID != pre.SequenceID {
		log.Info("消息序号对不上，拒绝执行commit广播")
	} else if !p.RsaVerySignWithSha256(digestByte, pre.Sign, MessageNodePubKey) {
		log.Info("节点签名验证失败！,拒绝执行commit广播")
	} else {
		p.setPrePareConfirmMap(pre.Digest, pre.NodeID, true)
		count := 0
		for range p.prePareConfirmCount[pre.Digest] {
			count++
		}
		//因为主节点不会发送Prepare，所以不包含自己
		specifiedCount := 0
		if p.node.nodeID == "N0" {
			specifiedCount = commonconst.NodeCount / 3 * 2
		} else {
			specifiedCount = (commonconst.NodeCount / 3 * 2) - 1
		}
		//如果节点至少收到了2f个prepare的消息（包括自己）,并且没有进行过commit广播，则进行commit广播
		p.lock.Lock()
		//获取消息源节点的公钥，用于数字签名验证
		if count >= specifiedCount && !p.isCommitBordcast[pre.Digest] {
			log.Info("本节点已收到至少2f个节点(包括本地节点)发来的Prepare信息 ...")

			//*******************************************************************

			//待注释--测试专用（节点作恶：即使全部验证通过也拒绝广播）
			//if p.node.nodeID=="N0"{
			//	log.Info("主节点作恶：全部验证通过，但是拒绝广播")
			//	p.lock.Unlock()
			//	return
			//}
			//
			//if p.node.nodeID=="N1"{
			//	log.Info("从节点N1作恶：全部验证通过，但是拒绝广播")
			//	p.lock.Unlock()
			//	return
			//}
			//
			//*******************************************************************

			//节点使用私钥对其签名
			sign := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
			c := Commit{pre.Digest, pre.SequenceID, p.node.nodeID, sign}
			bc, err := json.Marshal(c)
			if err != nil {
				log.Error(err)
			}
			//进行提交信息的广播
			log.Info("正在进行commit广播")
			msg := meta.TCPMessage{
				Type:    commonconst.PBFTCommit,
				Content: bc,
			}
			p.broadcast(msg)
			p.isCommitBordcast[pre.Digest] = true
			log.Info("commit广播完成")
		}
		p.lock.Unlock()
	}
}

//处理提交确认消息
func (p *pbft) handleCommit(content []byte) {
	//使用json解析出Commit结构体
	c := new(Commit)
	err := json.Unmarshal(content, c)
	if err != nil {
		log.Error(err)
	}
	log.Infof("本节点已接收到%s节点发来的Commit ... \n", c.NodeID)
	//获取消息源节点的公钥，用于数字签名验证
	MessageNodePubKey := p.getPubKey(c.NodeID)
	digestByte, _ := hex.DecodeString(c.Digest)
	if _, ok := p.prePareConfirmCount[c.Digest]; !ok {
		log.Info("当前prepare池无此摘要，拒绝将信息持久化到本地消息池")
	} else if p.sequenceID != c.SequenceID {
		log.Info("消息序号对不上，拒绝将信息持久化到本地消息池")
	} else if !p.RsaVerySignWithSha256(digestByte, c.Sign, MessageNodePubKey) {
		log.Info("节点签名验证失败！,拒绝将信息持久化到本地消息池")
	} else {
		p.setCommitConfirmMap(c.Digest, c.NodeID, true)
		count := 0
		for range p.commitConfirmCount[c.Digest] {
			count++
		}
		//如果节点至少收到了2f+1个commit消息（包括自己）,并且节点没有回复过,并且已进行过commit广播，则提交信息至本地消息池，并reply成功标志至客户端！
		p.lock.Lock()
		if count >= commonconst.NodeCount/3*2 && !p.isReply[c.Digest] && p.isCommitBordcast[c.Digest] {
			log.Info("本节点已收到至少2f + 1 个节点(包括本地节点)发来的Commit信息 ...")
			//将消息信息，提交到本地消息池中！
			localMessagePool = append(localMessagePool, p.messagePool[c.Digest].Message)
			info := p.node.nodeID + "节点已将msgid:" + strconv.Itoa(p.messagePool[c.Digest].ID) + "存入本地消息池中,消息内容为：" + p.messagePool[c.Digest].Content
			log.Info(info)
			//既然已经得到共识，新区块上链，落库
			bcs := chain.GetCurrentBlockChain()
			newBCMsg := p.messagePool[c.Digest].Content
			newBC := new(meta.Block)
			err = json.Unmarshal([]byte(newBCMsg), newBC)
			util.DealJsonErr("handleCommit", err)
			bcs = append(bcs, *newBC)
			chain.StoreBlockChain(bcs)
			//新区块上链后状态数据库进行更新
			p.refreshState(*newBC)
			//给客户端reply
			log.Info("正在reply客户端 ...")
			tcpMsg := meta.TCPMessage{
				Type:    commonconst.PBFTReply,
				Content: []byte(newBCMsg),
			}
			network.TCPSend(tcpMsg, p.messagePool[c.Digest].ClientAddr)
			p.isReply[c.Digest] = true
			log.Info("reply完毕")
		}
		p.lock.Unlock()
	}
}

//状态数据库更新
func (p *pbft) refreshState(b meta.Block) {
	//ste1：首先取出本区块中所有的交易
	txs:=b.TX
	//每一笔交易写集进行更新,若交易为部署新合约，则触发部署
	for _,tx:=range txs{
		if tx.To==commonconst.ContractDeployAddress && p.node.nodeID=="N0"{
			//部署合约
			contractName:=tx.Contract
			//生成build地址
			path:="./smart_contract/"+contractName+"/"
			smart_contract.BuildAndRun(path,contractName)
		}
		set:=tx.Data.Set
		for k,v:=range set{
			if k!=""{
				levelDB.DBPut(k,[]byte(v))
			}
		}
	}
}

//序号累加
func (p *pbft) sequenceIDAdd() {
	p.lock.Lock()
	p.sequenceID++
	p.lock.Unlock()
}

//向除自己外的其他节点进行广播
func (p *pbft) broadcast(msg meta.TCPMessage) {
	for i := range commonconst.NodeTable {
		if i == p.node.nodeID {
			continue
		}
		go network.TCPSend(msg, commonconst.NodeTable[i])
	}
}

//为多重映射开辟赋值
func (p *pbft) setPrePareConfirmMap(val, val2 string, b bool) {
	if _, ok := p.prePareConfirmCount[val]; !ok {
		p.prePareConfirmCount[val] = make(map[string]bool)
	}
	p.prePareConfirmCount[val][val2] = b
}

//为多重映射开辟赋值
func (p *pbft) setCommitConfirmMap(val, val2 string, b bool) {
	if _, ok := p.commitConfirmCount[val]; !ok {
		p.commitConfirmCount[val] = make(map[string]bool)
	}
	p.commitConfirmCount[val][val2] = b
}

//传入节点编号， 获取对应的公钥
func (p *pbft) getPubKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("Keys/" + nodeID + "/" + nodeID + "_RSA_PUB")
	if err != nil {
		log.Error(err)
	}
	return key
}

//传入节点编号， 获取对应的私钥
func (p *pbft) getPivKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("Keys/" + nodeID + "/" + nodeID + "_RSA_PIV")
	if err != nil {
		log.Error(err)
	}
	return key
}
