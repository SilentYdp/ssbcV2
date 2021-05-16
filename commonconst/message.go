package commonconst

const (
	//区块链同步（本链节点间）
	BlockSynReqMsg = "BlockSynReqMsg" //请求区块链同步消息
	BlockSynResMsg = "BlockSynResMsg" //区块链同步回复消息
)

const (
	//PBFT共识消息
	PBFT           = "PBFT"
	PBFTRequest    = "PBFTRequest"
	PBFTPrePrepare = "PBFTPrePrepare"
	PBFTPrepare    = "PBFTPrepare"
	PBFTCommit     = "PBFTCommit"
	PBFTReply      = "PBFTReply"
)
