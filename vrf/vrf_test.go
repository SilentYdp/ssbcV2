package vrf

import (
	"encoding/json"
	"fmt"
	"github.com/google/keytransparency/core/crypto/vrf/p256"
	"testing"
)

func TestVRF(t *testing.T) {
	//首先生成私钥k，公钥pk
	k, pk := p256.GenerateKey()
	m1 := []byte("data1")
	index1, proof1 := k.Evaluate(m1)

	pkByte, _ := json.Marshal(pk)
	pkStr := string(pkByte)

	var pubk p256.PublicKey

	err := json.Unmarshal([]byte(pkStr), &pubk)
	if err != nil {
		log.Info(err)
	}
	//验证生成的index
	index, err := pubk.ProofToHash(m1, proof1)
	log.Info("index=", index)
	if err != nil {
		log.Info("err=", err)
	}
	if got, want := index, index1; got != want {
		log.Info("验证失败")
	} else {
		log.Info("验证成功")
	}

}

func TestGenerateVrfResult(t *testing.T) {
	GenerateVrfResult("vrf")
}
