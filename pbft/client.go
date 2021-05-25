package pbft

import (
	"crypto/rand"
	"fmt"
	"github.com/ssbcV2/commonconst"
	"log"
	"math/big"
)

func ClientSendMessageAndListen() {
	//开始用户请求的监听
	go clientHttpListenV2()
	//开启客户端的本地监听（主要用来接收节点的reply信息）
	go clientTcpListen()
	fmt.Printf("客户端开启监听，地址：%s\n", commonconst.ClientToNodeAddr)
	fmt.Println(" ---------------------------------------------------------------------------------")
	fmt.Println("|  已进入PBFT客户端，请启动全部节点后再发送消息！  |")
	fmt.Println(" ---------------------------------------------------------------------------------")
}

//返回一个十位数的随机数，作为msgid
func getRandom() int {
	x := big.NewInt(10000000000)
	for {
		result, err := rand.Int(rand.Reader, x)
		if err != nil {
			log.Panic(err)
		}
		if result.Int64() > 1000000000 {
			return int(result.Int64())
		}
	}
}
