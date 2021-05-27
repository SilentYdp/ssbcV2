package pbft

import (
	"encoding/hex"
	"fmt"
	"github.com/ssbcV2/util"
	"testing"
)

func TestGenerateKey(t *testing.T)  {
	pri,pub:= GetKeyPair()
	log.Info("pri=",string(pri))
	log.Info("pub=",string(pub))
	//将公钥进行hash
	pubHash,_:=util.CalculateHash(pub)
	//将公钥的前20位作为账户地址
	account:=hex.EncodeToString(pubHash[:20])
	log.Info(account)
}

func TestGoModManage(t *testing.T) {
	_,errStr:=GoModManage("hellotest")
	log.Info("错误：",errStr)
}
