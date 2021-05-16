package chain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cloudflare/cfssl/log"
	"github.com/ssbcV2/commonconst"
	"github.com/ssbcV2/meta"
	"github.com/ssbcV2/redis"
)

//初始化的时候重置一次交易列表
func init() {
	ClearCurrentTxs()
}

//获取到当前交易集合列表
func GetCurrentTxs() []meta.Transaction {
	txs := make([]meta.Transaction, 0)
	txsStr, _ := redis.GetFromRedis(commonconst.TransActionsKey)
	err := json.Unmarshal([]byte(txsStr), &txs)
	if err != nil {
		fmt.Println("GetCurrentTxs:json unmarshal failed:", err)
	}
	return txs
}

//存储入当前交易
func StoreCurrentTx(tx meta.Transaction) {
	//首先获取到当前的交易列表
	curTxs := GetCurrentTxs()
	curTxs = append(curTxs, tx)
	txsByte, _ := json.Marshal(curTxs)
	txsStr := string(txsByte)
	redis.SetIntoRedis(commonconst.TransActionsKey, txsStr)
}

//重置当前交易列表
func ClearCurrentTxs() {
	txs := make([]meta.Transaction, 0)
	txsByte, _ := json.Marshal(txs)
	txsStr := string(txsByte)
	redis.SetIntoRedis(commonconst.TransActionsKey, txsStr)
}

//根据交易hash定位到所在区块的高度,以及该交易在交易列表中的序号
func LocateBlockHeightWithTran(transId []byte) (height int, sequence int) {
	//首先获取到当前全部的区块
	bcs := GetCurrentBlockChain()
	for h, bc := range bcs {
		//获取到当前区块所有的交易
		txs := bc.TX
		for sequence, tx := range txs {
			if bytes.Compare(tx.Id, transId) == 0 {
				return h, sequence
			}
		}
	}
	log.Error("未能定位到该笔交易")
	return -1, -1
}
