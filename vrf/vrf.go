package vrf

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/cloudflare/cfssl/log"
	"github.com/google/keytransparency/core/crypto/vrf/p256"
	"github.com/ssbcV2/commonconst"
	"github.com/ssbcV2/meta"
	"math"
)

//每个人都自己生成随机数，同时接受和验证他人的随机数，最小的那个有权发起提案

func VRF(count int) {
	//首先生成私钥k，公钥pk
	k, pk := p256.GenerateKey()
	m1 := []byte("data1")
	index1, proof1 := k.Evaluate(m1)
	log.Info("index1=", index1)
	log.Info("proof1=", proof1)
	tempIndex := make([]byte, 0)
	for i := 0; i < len(index1); i++ {
		tempIndex = append(tempIndex, index1[i])
	}
	bytebuff := bytes.NewBuffer(tempIndex)
	var data int64
	binary.Read(bytebuff, binary.BigEndian, &data)
	log.Info("data=", int(data))
	resultIndex := int64(math.Abs(float64(int(data)))) % int64(count)
	//验证生成的index
	log.Info("提案节点index=", resultIndex)
	index, err := pk.ProofToHash(m1, proof1)
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

func GenerateVrfResult(msg string) (result meta.VRFResult) {
	//首先生成私钥k，公钥pk
	k, pk := p256.GenerateKey()
	pkByte, _ := json.Marshal(pk)
	m1 := []byte(msg)
	index1, proof := k.Evaluate(m1)
	tempIndex := make([]byte, 0)
	for i := 0; i < len(index1); i++ {
		tempIndex = append(tempIndex, index1[i])
	}
	bytebuff := bytes.NewBuffer(tempIndex)
	var data int64
	err := binary.Read(bytebuff, binary.BigEndian, &data)
	if err != nil {
		log.Error(err)
	}
	r := math.Abs(float64(data))
	//log.Info("生成VRF结果:",r)
	log.Info("Generating Vrf Result:", r)
	res := meta.VRFResult{
		Result:      r,
		ResultIndex: index1,
		PK:          pkByte,
		Proof:       proof,
		Msg:         msg,
		Count:       commonconst.VRFThreshold,
	}
	return res
}

func VerifyVrf(result meta.VRFResult) bool {
	//首先公钥反序列化
	var pubk p256.PublicKey
	err := json.Unmarshal(result.PK, &pubk)
	if err != nil {
		//log.Info("[VerifyVrf] json unmarshal failed,err=", err)
	}
	//验证生成的index
	index, err := pubk.ProofToHash([]byte(result.Msg), result.Proof)
	if index == result.ResultIndex {
		log.Info("Vrf Result Verification Successful")
		return true
	} else {
		log.Info("Vrf Result Verification Failed")
		return false
	}
}
