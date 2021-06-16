package smart_contract

import (
	"encoding/json"
	"fmt"
	"github.com/cloudflare/cfssl/log"
	"testing"
)

func TestBuildAndRun(t *testing.T) {
	BuildAndRun("test/","test")
}
func TestGenerateCodeJson(t *testing.T) {
	a:= struct {
		Code string
	}{
		Code:`
package main

import (
"encoding/json"
"github.com/gorilla/mux"
"log"
"net/http"
"time"
)

/*
	一个合约部署的示例代码
*/

const NightMethod  = "night"
const ContractName  = "goodnight"//不要有大写字母


//这个固定的
type ContractResponse struct {
	Read map[string]string
	Set  map[string]string
}
//这个也是固定的
type ContractRequest struct {
	Method string
	Args   map[string]string
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
	case NightMethod:
		handleNightMethod(cr.Args,w)
	}

}
//args是一个键值对的map长下面这样
//{
//	"name":"ye"
//}

func handleNightMethod(args map[string]string,w http.ResponseWriter)  {
	day:=args["day"]
	result:="good"+day

	//根据传进来的参数如何处理自己可以发挥，但是处理结果交给链上处理得按如下的模板设置

	//下面的写法是固定的，理解：读集-就是不存到链上，就是调用这个合约函数看看结果，写集--需要更新到链上，存到链上。方便起见，可以设置为函数的调用结果
	//不需要存链，就放到读集里
	read:=make(map[string]string)
	read[day]=result
	res:=ContractResponse{
		Read: read,
		Set:  nil,
	}

	//这些都不用改
	resByte,_:=json.Marshal(res)
	w.Write(resByte)
}

//这块不用动
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
`}
	aB,_:=json.Marshal(a)
	fmt.Println(string(aB))
}

func TestGenerateArgsJsonMap(t *testing.T){
	m:=make(map[string]string)
	m["day"]="monday"
	m["year"]="22"
	mB,_:=json.Marshal(m)
	a:= struct {
		Args string
	}{
		Args:string(mB),
	}
	aB,_:=json.Marshal(a)
	log.Info(string(aB))
}



func TestListAllContains(t *testing.T) {
	cs:=ListAllContains()
	for _, container := range cs {
		log.Info(container.ID, container.Image)
		for _,p:=range container.Ports{
			log.Info(p.PublicPort)
		}
	}

}


func TestStrToMap(t *testing.T){
	s:="{\"dest\":\"ssbc2\",\"height\":\"0\"}"
	m:=make(map[string]string)

	err:=json.Unmarshal([]byte(s),&m)
	log.Info(err)
	log.Info(m)
}

