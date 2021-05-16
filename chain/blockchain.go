package chain

import (
	"encoding/json"
	"fmt"
	"github.com/cloudflare/cfssl/log"
	"github.com/ssbcV2/commonconst"
	"github.com/ssbcV2/levelDB"
	"github.com/ssbcV2/merkle"
	"github.com/ssbcV2/meta"
	"github.com/ssbcV2/util"
	"sync"
	"time"
)

var mutex = &sync.Mutex{}

var BlockChain []meta.Block //声明全局变量

//生成创世区块
func GenerateGenesisBlock() meta.Block {
	genesisBlock := meta.Block{}
	genesisBlock = meta.Block{
		Timestamp: time.Now().String(),
	}
	genesisBlock.Hash = util.CalculateBlockHash(genesisBlock)
	return genesisBlock
}

//将当前区块链存储
func StoreBlockChain(bc []meta.Block) {
	bcBytes, _ := json.Marshal(bc)
	//缓存存储
	//redis.SetIntoRedis(commonconst.BlockChainKey, string(bcBytes))
	//db存储
	levelDB.DBPut(commonconst.BlockChainKey, bcBytes)
}

//获取到当前区块链
func GetCurrentBlockChain() []meta.Block {
	//bcStr, _ := redis.GetFromRedis(commonconst.BlockChainKey)
	//更新版本从db中读取区块链
	bcByte := levelDB.DBGet(commonconst.BlockChainKey)
	bc := make([]meta.Block, 0)
	if bcByte == nil {
		return bc
	}
	err := json.Unmarshal(bcByte, &bc)
	if err != nil {
		log.Errorf("[GetCurrentBlockChain] unmarshal failed:err=%v", err)
	}
	return bc
}

//获取到全部的交易
func GetAllTransactions() []meta.Transaction {
	//先获取到当前的区块链
	bc := GetCurrentBlockChain()
	//取出所有的交易信息
	allTrans := make([]meta.Transaction, 0)
	for _, b := range bc {
		t := b.TX
		allTrans = append(allTrans, t...)
	}
	return allTrans
}

//根据区块高度获取到具体的区块
func GetBlock(index int) *meta.Block {
	bcs := GetCurrentBlockChain()
	if index >= len(bcs) {
		log.Error("[GetBlock],区块高度参数非法，当前区块高度：", len(bcs))
		return nil
	}
	bc := bcs[index]
	return &bc
}

//生成新区块
func GenerateNewBlock(txs []meta.Transaction) meta.Block {
	//首先获取到当前的区块链
	curBlockChain := GetCurrentBlockChain()
	length := len(curBlockChain)
	preBlock := curBlockChain[length-1]
	var newBlock = meta.Block{
		Height:    len(curBlockChain),
		Timestamp: time.Now().String(),
		PrevHash:  preBlock.Hash,
		TX:        txs,
	}
	//生成该区块的merkle root
	//区块中的每一笔交易生成对应的merkle path、index
	tree := merkle.GenerateMerkleTree(txs)
	newBlock.MerkleRoot = tree.MerkleRoot()
	newBlock.Hash = util.CalculateBlockHash(newBlock)
	return newBlock
}

//同步其他节点发来的区块链，进行本地更新
func UpdateChain(bc []meta.Block) {
	//首先获取本地的chain
	bcLocal := GetCurrentBlockChain()
	if len(bcLocal) < len(bc) {
		//更新
		StoreBlockChain(bc)
		log.Info("更新区块链成功")
		bytes, err := json.MarshalIndent(bc, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		// Green console color: 	\x1b[32m
		// Reset console color: 	\x1b[0m
		fmt.Printf("\x1b[32m%s\x1b[0m> ", string(bytes))
		fmt.Println()
	}
}

//打印更新后的区块链
func PrintUpdatedBlockChain(bc []meta.Block) {
	bytes, err := json.MarshalIndent(bc, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	// Green console color: 	\x1b[32m
	// Reset console color: 	\x1b[0m
	fmt.Printf("\x1b[32m%s\x1b[0m> ", string(bytes))
}
