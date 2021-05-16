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
	"net/http"
	"strconv"
	"time"
)

type Server struct {
	Url string
}

func NewClientServer(url string) *Server {
	server := &Server{Url: url}
	server.setRoute()
	return server
}
func (server *Server) Start() {
	fmt.Printf("Server will be started at %s...\n", server.Url)
	if err := http.ListenAndServe(server.Url, nil); err != nil {
		fmt.Println(err)
		return
	}
}

//跨域处理
func cors(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")                                                            // 允许访问所有域，可以换成具体url，注意仅具体url才能带cookie信息
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token") //header的类型
		w.Header().Add("Access-Control-Allow-Credentials", "true")                                                    //设置为true，允许ajax异步请求带cookie信息
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")                             //允许请求方法
		w.Header().Set("content-type", "application/json;charset=UTF-8")                                              //返回数据格式是json
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		f(w, r)
	}
}

func (server *Server) setRoute() {
	//获取到区块列表
	http.HandleFunc("/getBlockChain", cors(server.getBlockChain))
	//获取到指定高度的区块
	http.HandleFunc("/getBlock", cors(server.getBlock))
	//获取到指定高度区块的交易列表
	http.HandleFunc("/getOneBlockTrans", cors(server.getBlockTrans))
	//提交一笔交易
	http.HandleFunc("/postTran", cors(server.postTran))
	//获取全部交易
	http.HandleFunc("/getAllTrans", cors(server.getAllTrans))
	//提交一笔跨链交易
	//http.HandleFunc("/postCrossTran", server.postCrossTran)
	//提交智能合约
	//http.HandleFunc("/postContract", server.postContract)
	//提供链上query服务--既能服务于普通节点也能服务于智能合约
	http.HandleFunc("/query", cors(server.query))

}

//链上信息query服务
func (server *Server) query(writer http.ResponseWriter, request *http.Request) {
	//测试用，之后需要删掉(库也需要删)
	tests := make([]meta.AbstractBlockHeader, 0)
	test := meta.AbstractBlockHeader{
		ChainId:    "ssbc2",
		Height:     0,
		Hash:       []byte("hello"),
		PreHash:    []byte("hello"),
		MerkleRoot: []byte("hello"),
	}
	tests = append(tests, test)
	tests = append(tests, test)
	testKey := "abstract_block_header_store_key_ssbc2"
	testsByte, _ := json.Marshal(tests)
	levelDB.DBPut(testKey, testsByte)

	//获取到查询参数
	queryKey := network.ParseGetParam("queryKey", request)
	//根据查询key去库中查询数据
	val := levelDB.DBGet(queryKey)
	warpHttpResponse(writer, val)
}

//获取全部的交易
func (server *Server) getAllTrans(writer http.ResponseWriter, request *http.Request) {
	all := chain.GetAllTransactions()
	warpHttpResponse(writer, all)
}

//提交一笔交易
func (server *Server) postTran(writer http.ResponseWriter, request *http.Request) {
	t := meta.Transaction{}
	err := json.NewDecoder(request.Body).Decode(&t)
	if err != nil {
		log.Error("[postTran],json decode err:", err)
	}
	//客户端需要把交易信息发送给主节点
	r := new(Request)
	r.Timestamp = time.Now().UnixNano()
	r.ClientAddr = clientAddr
	r.Message.ID = getRandom()

	tb, _ := json.Marshal(t)
	r.Message.Content = string(tb)
	br, err := json.Marshal(r)
	if err != nil {
		log.Error(err)
	}
	fmt.Println(string(br))
	msg := meta.TCPMessage{
		Type:    commonconst.PBFTRequest,
		Content: br,
	}
	//默认N0为主节点，直接把请求信息发送至N0
	network.TCPSend(msg, commonconst.NodeTable["N0"])
}

//用户查询当前所有区块-->获取当前的区块链
func (server *Server) getBlockChain(writer http.ResponseWriter, request *http.Request) {
	//获取当前区块链
	bcs := chain.GetCurrentBlockChain()
	warpHttpResponse(writer, bcs)
}

//用户根据区块高度获取到某一个区块
func (server *Server) getBlock(writer http.ResponseWriter, request *http.Request) {
	//获取到请求参数中的区块高度
	h := network.ParseGetParam("height", request)
	hInt64, err := strconv.ParseInt(h, 10, 64)
	if err != nil {
		log.Error("[getBlock],parseInt err:", err)
		panic(err)
	}
	hInt := int(hInt64)
	bc := chain.GetBlock(hInt)
	if bc == nil {
		network.BadRequestResponse(writer)
		return
	}
	warpHttpResponse(writer, bc)
}

//用户获取到某一区块中的所有交易
func (server *Server) getBlockTrans(writer http.ResponseWriter, request *http.Request) {
	//先解析出请求中的区块高度
	h := network.ParseGetParam("height", request)
	hInt64, err := strconv.ParseInt(h, 10, 64)
	if err != nil {
		log.Error("[getBlock],parseInt err:", err)
		panic(err)
	}
	hInt := int(hInt64)
	bc := chain.GetBlock(hInt)
	if bc == nil {
		network.BadRequestResponse(writer)
		return
	}
	trans := bc.TX
	warpHttpResponse(writer, trans)
}

func warpHttpResponse(writer http.ResponseWriter, data interface{}) {
	res := meta.HttpResponse{
		StatusCode: http.StatusOK,
		Data:       data,
	}
	b, err := json.Marshal(res)
	util.DealJsonErr("warpHttpResponse", err)
	writer.Write(b)
}

//假区块用以测试
func FakeBlockChain() []meta.Block {
	bcs := make([]meta.Block, 0)
	testSig, _ := util.CalculateHash([]byte("Signature"))
	testHash, _ := util.CalculateHash([]byte("Hash"))
	testMerkleRoot, _ := util.CalculateHash([]byte("MerkleRoot"))
	Txs := make([]meta.Transaction, 0)
	T1 := FakeTransaction()
	Txs = append(Txs, T1)
	Txs = append(Txs, T1)
	bc1 := meta.Block{
		Height:     0,
		Timestamp:  time.Now().String(),
		PrevHash:   nil,
		MerkleRoot: nil,
		Signature:  nil,
		Hash:       testHash,
		TX:         nil,
	}
	bc2 := meta.Block{
		Height:     1,
		Timestamp:  time.Now().String(),
		PrevHash:   testHash,
		MerkleRoot: testMerkleRoot,
		Signature:  testSig,
		Hash:       testHash,
		TX:         Txs,
	}
	bcs = append(bcs, bc1)
	bcs = append(bcs, bc2)
	return bcs
}

func FakeTransaction() meta.Transaction {
	testID, _ := util.CalculateHash([]byte("ID"))
	t := meta.Transaction{
		From:  "A",
		To:    "B",
		Data:  meta.TransactionData{},
		Value: 10,
		Id:    testID,
	}
	return t
}
