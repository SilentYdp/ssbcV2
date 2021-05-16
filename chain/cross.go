package chain

import (
	"encoding/json"
	"github.com/ssbcV2/commonconst"
	"github.com/ssbcV2/merkle"
	"github.com/ssbcV2/meta"
	"github.com/ssbcV2/util"
	"time"
)

//获取到本链的抽象区块头集合
func GetLocalAbstractBlockChainHeaders(chainId string) []meta.AbstractBlockHeader {
	//首先获取到本链区块链
	bc := GetCurrentBlockChain()
	bcHeaders := make([]meta.AbstractBlockHeader, 0)
	for h, block := range bc {
		var bcHeader meta.AbstractBlockHeader
		bcHeader = meta.AbstractBlockHeader{
			ChainId:    chainId,
			Height:     h,
			Hash:       block.Hash,
			PreHash:    block.PrevHash,
			MerkleRoot: block.MerkleRoot,
		}
		bcHeaders = append(bcHeaders, bcHeader)
	}
	return bcHeaders
}

//打包跨链交易回执
func PackCrossReceipt(t meta.CrossTran, height int, sequence int) meta.CrossTranReceipt {
	//首先根据区块高度获取到指定的区块
	cBcs := GetCurrentBlockChain()
	bc := cBcs[height]
	//获取到区块中所有的交易
	txs := bc.TX
	//生成该交易的merkle proof
	tranHash, merklePath, merkleIndex := merkle.GetTranMerklePath(txs, sequence)
	var re meta.CrossTranReceipt
	uc, _ := json.Marshal(*util.LocalPublicKey)

	//进行签名
	sign := util.RSASign(tranHash, util.LocalPrivateKey)
	resp := meta.CrossTranResp{
		Data:      nil,
		Signature: sign,
	}
	pf := meta.CrossTranProof{
		MerklePath:  merklePath,
		TransHash:   tranHash,
		Height:      height,
		MerkleIndex: merkleIndex,
	}
	re = meta.CrossTranReceipt{
		SourceChainId:   t.DestChainId,
		DestChainId:     t.SourceChainId,
		TimeStamp:       time.Now().String(),
		TimeOut:         time.Now().Add(time.Hour).String(),
		UserCertificate: uc,
		TransId:         txs[sequence].Id,
		Type:            t.Type,
		Status:          commonconst.StatusSuccess,
		Resp:            resp,
		Proof:           pf,
	}
	return re
}

//打包一笔跨链交易
func PackACrossTransaction(t meta.CrossTran, height int, sequence int) meta.CrossTran {
	//首先根据区块高度获取到指定的区块
	cBcs := GetCurrentBlockChain()
	bc := cBcs[height]
	//获取到区块中所有的交易
	txs := bc.TX
	//生成该交易的merkle proof
	tranHash, merklePath, merkleIndex := merkle.GetTranMerklePath(txs, sequence)
	var ct meta.CrossTran
	uc, _ := json.Marshal(*util.LocalPublicKey)
	pf := meta.CrossTranProof{
		MerklePath:  merklePath,
		TransHash:   tranHash,
		Height:      height,
		MerkleIndex: merkleIndex,
	}
	ct = meta.CrossTran{
		SourceChainId:   t.SourceChainId,
		DestChainId:     t.DestChainId,
		TimeStamp:       time.Now().String(),
		TimeOut:         time.Now().Add(time.Hour).String(),
		UserCertificate: uc,
		TransId:         txs[sequence].Id,
		Type:            t.Type,
		Status:          commonconst.StatusDeal,
		Param:           meta.CrossTranParam{},
		Proof:           pf,
	}

	return ct
}
