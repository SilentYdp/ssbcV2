package main

import (
	"encoding/hex"
	"fmt"
	"github.com/ssbcV2/util"
	"testing"
)

func TestGetBlockChain(t *testing.T) {
	s := NewClientServer(clientHttpAddr)
	s.Start()
}

func TestGenerateKey(t *testing.T)  {
	pri,pub:=getKeyPair()
	fmt.Println("pri=",string(pri))
	fmt.Println("pub=",string(pub))
	//将公钥进行hash
	pubHash,_:=util.CalculateHash(pub)
	//将公钥的前20位作为账户地址
	account:=hex.EncodeToString(pubHash[:20])
	fmt.Println(account)
}
