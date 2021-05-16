package util

import "github.com/cloudflare/cfssl/log"

func DealJsonErr(funcName string, err error) {
	if err != nil {
		log.Error("["+funcName+"]"+",json marshal or unmarshal failed.err:", err)
	}
}
