package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

/*
	抽象区块头存储合约
	主要函数：
1.存储最新的抽象区块头
2.获取指定高度的抽象区块头
3.获取抽象区块头同步策略
4.设置抽象区块头同步策略
*/

const AbstractBlockHeaderStoreKey  = "abstract_block_header_store_key_"
const GetAbstractBlockHeader  = "GetAbstractBlockHeader"
const ContractName  = "abhs"


type AbstractBlockHeader struct {
	ChainId    string
	Height     int
	Hash       []byte
	PreHash    []byte
	MerkleRoot []byte
}

type QueryResp struct {
	StatusCode int
	Data       []byte
}

type ContractResponse struct {
	Read map[string]string
	Set  map[string]string
}
type ContractRequest struct {
	Method string
	Args   map[string]string
}

func getAbstractBlockHeader(h int,dest string) *AbstractBlockHeader {
	//首先获取到列表
	abs:=getAbstractBlockHeaders(dest)
	//再根据指定高度取的区块头
	for _,header:=range abs{
		if header.Height==h{
			return &header
		}
	}
	//找不到的话返回空
	return nil
}

func StoreAbstractBlockHeader(header string,dest string) ContractResponse {
	setMap:=make(map[string]string)
	h:=AbstractBlockHeader{}
	err:=json.Unmarshal([]byte(header),&h)
	if err!=nil{
		log.Println("[StoreAbstractBlockHeader] json unmarshal failed,err",err)
	}
	//先获取到当前存储的抽象区块头列表
	abs:=getAbstractBlockHeaders(dest)
	//将新区块头附加进去
	abs=append(abs,h)
	absByte,_:=json.Marshal(abs)
	key:=AbstractBlockHeaderStoreKey+dest
	setMap[key]=string(absByte)
	res:=ContractResponse{
		Read: nil,
		Set:  setMap,
	}
	return res
}

//获取到抽象区块头列表
func getAbstractBlockHeaders(dest string) []AbstractBlockHeader {
	//先拼装查询key
	key:=AbstractBlockHeaderStoreKey+dest
	//调用智能合约数据服务
	params := url.Values{}
	Url, err := url.Parse("http://docker.for.mac.host.internal:9999/query")
	if err!=nil{
		log.Println("url parse err:",err)
	}

	params.Set("queryKey",key)
	//如果参数中有中文参数,这个方法会进行URLEncode
	Url.RawQuery = params.Encode()
	urlPath := Url.String()

	resp,err := http.Get(urlPath)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	res:=QueryResp{}
	err=json.Unmarshal(body,&res)

	abs:=make([]AbstractBlockHeader,0)
	err=json.Unmarshal(res.Data,&abs)
	if err!=nil{
		log.Println("[getAbstractBlockHeaders] json unmarshal failed,err",err)
	}
	return abs
}

func handleGetAbstractBlockHeader(args map[string]string,w http.ResponseWriter)  {
	h:=args["height"]
	hInt64, err := strconv.ParseInt(h, 10, 64)
	if err != nil {
		log.Println("[handleGetAbstractBlockHeader],parseInt err:", err)
	}
	hInt := int(hInt64)
	dest:=args["dest"]

	ab:=getAbstractBlockHeader(hInt,dest)
	abStr,_:=json.Marshal(ab)

	read:=make(map[string]string)
	read["abh"]=string(abStr)
	res:=ContractResponse{
		Read: read,
		Set:  nil,
	}

	resByte,_:=json.Marshal(res)
	w.Write(resByte)
}


func handler(w http.ResponseWriter, r *http.Request) {
	log.Println("接收到请求:",r.Body)
	cr := ContractRequest{}
	err := json.NewDecoder(r.Body).Decode(&cr)
	if err!=nil{
		log.Println(err)
	}
	methodName:=cr.Method
	switch methodName {
	case GetAbstractBlockHeader:
		handleGetAbstractBlockHeader(cr.Args,w)
	}

}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", handler)
	server := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Println("Starting Server v6.")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
