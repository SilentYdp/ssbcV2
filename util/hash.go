package util

import (
	"crypto/sha256"
	"encoding/json"
	"github.com/cloudflare/cfssl/log"
	"github.com/ssbcV2/meta"
)

//计算hash摘要
func CalculateHash(msg []byte) ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write(msg); err != nil {
		log.Info(err)
		return nil, err
	}
	return h.Sum(nil), nil
}

//计算区块hash
func CalculateBlockHash(b meta.Block) []byte {
	jb, _ := json.Marshal(b)
	hashed, _ := CalculateHash(jb)
	return hashed
}
