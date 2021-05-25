package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/ssbcV2/docker"
	"log"
	"net/http"
	"time"
)

/*
跨链注册、链名解析合约
主要函数：
1.跨链注册
2.链名解析
*/

type ContractRequest struct {
	Method string
	Args   map[string]string
}
type ContractResponse struct {
	Read map[string]string
	Set  map[string]string
}
type ChainMeta struct {
	Name string
	Relayers  []string
	Servers   []string
}

const (
	Register  = "cross_register"  //跨链注册函数名
	Parse     = "cross_parse" 	 //链名解析
	ContractName="cross"   //合约名
	)


func handler(w http.ResponseWriter, r *http.Request) {
	log.Println("接收到请求:",r.Body)
	cr := ContractRequest{}
	err := json.NewDecoder(r.Body).Decode(&cr)
	if err!=nil{
		log.Println(err)
	}
	methodName:=cr.Method
	switch methodName {
	case Register:
		handleRegister(cr.Args,w)
	case Parse:
		handleParse(cr.Args,w)
	}
}
func handleParse(args map[string]string,w http.ResponseWriter) {
	dest:=args["name"]
	key:=generateStoreKey(dest)
	//调用数据服务获取chainMeta
	data:=docker.ContractDataServer(key)
	read:=make(map[string]string)
	read[key]=string(data)
	res:=ContractResponse{
		Read: read,
		Set:  nil,
	}
	resByte,_:=json.Marshal(res)
	w.Write(resByte)
}

func handleRegister(args map[string]string,w http.ResponseWriter)  {
	dest:=args["name"]
	relayersStr:=args["relayer"]
	serversStr:=args["servers"]

	//是数组
	relayers:=make([]string,0)
	err:=json.Unmarshal([]byte(relayersStr),&relayers)
	if err!=nil{
		log.Println(err)
	}
	servers:=make([]string,0)
	err=json.Unmarshal([]byte(serversStr),&servers)
	if err!=nil{
		log.Println(err)
	}

	set:=make(map[string]string)
	key:=generateStoreKey(dest,Register)

	meta:=ChainMeta{
		Name:     dest,
		Relayers: relayers,
		Servers:  servers,
	}
	metaByte,_:=json.Marshal(meta)
	set[key]=string(metaByte)
	res:=ContractResponse{
		Read: nil,
		Set:  set,
	}
	resByte,_:=json.Marshal(res)
	w.Write(resByte)
}

func generateStoreKey(k string) string {
	key:=ContractName+"_"+k
	return key
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
	log.Println("Starting Server ")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
