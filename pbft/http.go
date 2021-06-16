package pbft

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/cloudflare/cfssl/log"
	"github.com/gin-gonic/gin"
	"github.com/ssbcV2/chain"
	"github.com/ssbcV2/commonconst"
	"github.com/ssbcV2/levelDB"
	"github.com/ssbcV2/meta"
	"github.com/ssbcV2/network"
	"github.com/ssbcV2/util"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

//5.21-版本升级web服务使用gin框架
func clientHttpListenV2()  {
	r:=gin.Default()
	//使用跨域组件
	r.Use(Cors())
	//获取到当前区块链
	r.GET("/getBlockChain",getBlockChain)
	//获取到指定高度的区块
	r.GET("/getBlock", getBlock)
	//获取到指定高度区块的交易列表
	r.GET("/getOneBlockTrans", getBlockTrans)
	//提交一笔交易
	r.POST("/postTran", postTran)
	//获取全部交易
	r.GET("/getAllTrans", getAllTrans)
	//注册账户
	r.GET("/registerAccount",registerAccount)
	//提交一笔跨链交易
	//http.HandleFunc("/postCrossTran", server.postCrossTran)
	//提交智能合约
	r.POST("/postContract", postContract)
	//提供链上query服务--既能服务于普通节点也能服务于智能合约
	r.GET("/query", query)
	//r.POST("/decrypt",decrypt)

	r.Run(commonconst.ClientToUserAddr)
}





func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		origin := c.Request.Header.Get("Origin")

		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization") //自定义 Header
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")

		}

		if method == "OPTIONS" {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization") //自定义 Header
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.AbortWithStatus(http.StatusNoContent)
		}

		c.Next()
	}
}


//提交智能合约代码
func postContract (ctx *gin.Context){
	// 读取此次提交
	postC:=meta.ContractPost{}
	_ = ctx.ShouldBind(&postC)
	//得先获取到合约名
	contractName:=postC.Name
	//先在docker文件目录中创建合约文件夹
	if isExist("./smart_contract/"+contractName){
		log.Error("该合约已存在")
		hr:=warpBadHttpResponse("同名合约已存在")
		ctx.JSON(http.StatusBadRequest,hr)
	}else {
		err:=os.Mkdir("./smart_contract/"+contractName,0777)
		if err!=nil{
			log.Error(err)
		}
		// 创建保存文件
		destFile, err := os.Create("./smart_contract/"+contractName+"/" + contractName+".go")
		if err != nil {
			log.Error("Create failed: %s\n", err)
			return
		}
		defer destFile.Close()
		_, _ = destFile.WriteString(postC.Code)

		//创建Dockfile文件
		GenerateDockerFile(contractName)
		//解决代码依赖问题
		err,errStr:=GoModManage(contractName)
		if err!=nil{
			//将文件夹删除
			//err:=os.RemoveAll("./smart_contract/"+contractName)
			//if err!=nil{
			//	log.Error(err)
			//}
			hr:=warpBadHttpResponse(errStr)
			log.Error(err)
			ctx.JSON(http.StatusBadRequest,hr)
		}else {
			//除了返回发送成功外，需要将此部署封装为交易发送至主节点，经共识后真正部署
			go sendNewContract(postC)
			hr:=warpGoodHttpResponse("SuccessFully")
			ctx.JSON(http.StatusOK,hr)
		}
	}
}

//解决Dockerfile
func GenerateDockerFile(path string)  {
	df,err:=os.Create("./smart_contract/"+path+"/" + "Dockerfile")
	if err != nil {
		log.Error(err)
	}
	defer df.Close()
	source, err := os.Open("./smart_contract/Dockerfile")
	if err != nil {
		log.Error(err)
	}
	defer source.Close()

	_, err = io.Copy(df, source)
	if err != nil {
		log.Error(err)
	}
}

//将部署封装为交易发送至主节点
func sendNewContract(c meta.ContractPost)  {
	data:=meta.TransactionData{}
	data.Code=c.Code
	t:=meta.Transaction{
		From:       c.Account,
		To:         commonconst.ContractDeployAddress,
		Dest:       "",
		Contract:   c.Name,
		Method:     "",
		Args:       nil,
		Data:       data,
		Value:      0,
		Id:         nil,
		Timestamp:  "",
		Hash:       nil,
		PublicKey:  c.PublicKey,
		Sign:       nil,
	}
	//客户端在转发交易之前需要对交易进行签名
	//先将交易进行hash
	tByte,_:=json.Marshal(t)
	t.Hash,_=util.CalculateHash(tByte)
	t.Sign=RsaSignWithSha256(t.Hash,[]byte(c.PrivateKey))
	//客户端需要把交易信息发送给主节点
	r := new(Request)
	r.Timestamp = time.Now().UnixNano()
	r.ClientAddr = commonconst.ClientToNodeAddr
	r.Message.ID = getRandom()

	tb, _ := json.Marshal(t)
	r.Message.Content = string(tb)
	br, err := json.Marshal(r)
	if err != nil {
		log.Error(err)
	}
	//log.Info(string(br))
	msg := meta.TCPMessage{
		Type:    commonconst.PBFTRequest,
		Content: br,
	}
	//默认N0为主节点，直接把请求信息发送至N0
	network.TCPSend(msg, commonconst.NodeTable["N0"])
}

func GoModManage(contractName string)(err error,errStr string)  {
	var output1,output2,output3 bytes.Buffer
	//执行依赖管理指令
	cmd:=exec.Command("go","mod","init")
	cmd.Dir="./smart_contract/"+contractName
	cmd.Stderr=&output1
	err=cmd.Run()
	if err!=nil{
		log.Error(err)
		return err,output1.String()
	}else {
		log.Info(output1.String())
	}

	cmd=exec.Command("go","mod","tidy")
	cmd.Dir="./smart_contract/"+contractName
	cmd.Stdout=&output2
	err=cmd.Run()
	if err!=nil{
		log.Error(err)
		return err,output2.String()
	}else {
		log.Info(output2.String())
	}

	//执行编译命令
	cmd=exec.Command("go","build")
	cmd.Dir="./smart_contract/"+contractName
	cmd.Stderr=&output3
	err=cmd.Run()
	if err!=nil{
		log.Info(output3.String())
		log.Error(err)
		return err,output3.String()
	}else {
		log.Info(output3.String())
	}

	return nil,""
}

//账户注册
func  registerAccount (ctx *gin.Context){
	//首先生成公私钥
	priKey,PubKey:= GetKeyPair()
	//账户地址
	//将公钥进行hash
	pubHash,_:=util.CalculateHash(PubKey)
	//将公钥的前20位作为账户地址
	account:=hex.EncodeToString(pubHash[:20])
	res:= struct {
		PrivateKey string
		PublicKey  string
		AccountAddress string
	}{
		string(priKey),
		string(PubKey),
		account,
	}
	hr:= warpGoodHttpResponse(res)
	ctx.JSON(http.StatusOK,hr)
}

//链上信息query服务
func  query(ctx *gin.Context) {
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
	queryKey := ctx.Query("queryKey")
	//根据查询key去库中查询数据
	val := levelDB.DBGet(queryKey)
	log.Info("链上数据服务查询结果:",string(val))
	hr:= warpGoodHttpResponse(val)
	ctx.JSON(http.StatusOK,hr)
}

//获取全部的交易
func  getAllTrans(ctx *gin.Context) {
	all := chain.GetAllTransactions()
	hr:= warpGoodHttpResponse(all)
	ctx.JSON(http.StatusOK,hr)
}

//提交一笔交易
func  postTran(ctx *gin.Context) {
	pt := meta.PostTran{}
	err := ctx.ShouldBind(&pt)
	if err != nil {
		log.Error("[postTran],json decode err:", err)
	}
	//将args解析
	args:=make(map[string]string)
	err=json.Unmarshal([]byte(pt.Args),&args)
	if err!=nil{
		log.Error("[postTran] json err:",err)
	}
	t:=meta.Transaction{
		From:      pt.From,
		To:        pt.To,
		Dest:      pt.Dest,
		Contract:  pt.Contract,
		Method:    pt.Method,
		Args:      args,
		Data:      meta.TransactionData{},
		Value:     pt.Value,
		Id:        nil,
		Timestamp: "",
		Hash:      nil,
		PublicKey: pt.PublicKey,
		Sign:      nil,
	}
	//客户端在转发交易之前需要对交易进行签名
	//先将交易进行hash
	tByte,_:=json.Marshal(t)
	t.Hash,_=util.CalculateHash(tByte)
	t.Sign=RsaSignWithSha256(t.Hash,[]byte(pt.PrivateKey))
	//客户端需要把交易信息发送给主节点
	r := new(Request)
	r.Timestamp = time.Now().UnixNano()
	r.ClientAddr = commonconst.ClientToNodeAddr
	r.Message.ID = getRandom()

	tb, _ := json.Marshal(t)
	r.Message.Content = string(tb)
	br, err := json.Marshal(r)
	if err != nil {
		log.Error(err)
	}
	//log.Info(string(br))
	msg := meta.TCPMessage{
		Type:    commonconst.PBFTRequest,
		Content: br,
	}
	//默认N0为主节点，直接把请求信息发送至N0
	network.TCPSend(msg, commonconst.NodeTable["N0"])
	//返回提交成功
	hr:= warpGoodHttpResponse("Post Successfully!")
	ctx.JSON(http.StatusOK,hr)
}

//func decrypt (ctx *gin.Context){
//	type Decrypt struct {
//		PublicKey string
//		CipherText string
//	}
//
//	d:=Decrypt{}
//	_:=ctx.ShouldBind(&d)
//
//	util.RSADecrypt([]byte(d.CipherText),[]byte(d.PublicKey))
//
//
//}

//用户查询当前所有区块-->获取当前的区块链
func  getBlockChain(ctx *gin.Context) {
	//获取当前区块链
	bcs := chain.GetCurrentBlockChain()
	hr:= warpGoodHttpResponse(bcs)
	ctx.JSON(http.StatusOK,hr)
}

//用户根据区块高度获取到某一个区块
func  getBlock(ctx *gin.Context) {
	//获取到请求参数中的区块高度
	h := ctx.Query("height")
	hInt64, err := strconv.ParseInt(h, 10, 64)
	if err != nil {
		log.Error("[getBlock],parseInt err:", err)
		panic(err)
	}
	hInt := int(hInt64)
	bc := chain.GetBlock(hInt)
	if bc == nil {
		hr:= warpGoodHttpResponse("Invalid param")
		ctx.JSON(http.StatusBadRequest,hr)
	}else {
		hr:= warpGoodHttpResponse(bc)
		ctx.JSON(http.StatusOK,hr)
	}
}

//用户获取到某一区块中的所有交易
func  getBlockTrans(ctx *gin.Context) {
	//先解析出请求中的区块高度
	h := ctx.Query("height")
	hInt64, err := strconv.ParseInt(h, 10, 64)
	if err != nil {
		log.Error("[getBlock],parseInt err:", err)
		panic(err)
	}
	hInt := int(hInt64)
	bc := chain.GetBlock(hInt)
	if bc == nil {
		hr:= warpGoodHttpResponse("Invalid param")
		ctx.JSON(http.StatusBadRequest,hr)
	}else {
		trans := bc.TX
		hr:= warpGoodHttpResponse(trans)
		ctx.JSON(http.StatusOK,hr)
	}
}


func warpGoodHttpResponse(data interface{})meta.HttpResponse {
	res := meta.HttpResponse{
		StatusCode: http.StatusOK,
		Data:       data,
	}
	return res
}
func warpBadHttpResponse(data interface{})meta.HttpResponse {
	res := meta.HttpResponse{
		StatusCode: http.StatusBadRequest,
		Data:       data,
	}
	return res
}

func warpHttpResponse(status int,data interface{}) meta.HttpResponse {
	res := meta.HttpResponse{
		StatusCode: status,
		Data:       data,
	}
	return res
}



