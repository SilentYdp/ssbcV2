package meta

//抽象区块头
type AbstractBlockHeader struct {
	ChainId    string
	Height     int
	Hash       []byte
	PreHash    []byte
	MerkleRoot []byte
}

//跨链交易
type CrossTran struct {
	SourceChainId   string //源链链名
	DestChainId     string //目标链链名
	TimeStamp       string
	TimeOut         string //超时设置
	UserCertificate []byte //交易发起用户证书
	TransId         []byte //交易Id
	Type            string //交易类型
	Status          string //交易状态
	From            string //源账户
	To              string //目标账户
	Value           int
	Param           CrossTranParam
	Proof           CrossTranProof
}

//跨链回执
type CrossTranReceipt struct {
	SourceChainId   string //源链链名
	DestChainId     string //目标链链名
	TimeStamp       string
	TimeOut         string        //超时设置
	UserCertificate []byte        //交易发起用户证书
	TransId         []byte        //交易Id
	Type            string        //交易类型
	Status          string        //交易状态
	Resp            CrossTranResp //回执
	Proof           CrossTranProof
}

type CrossTranResp struct {
	Data      []byte //响应结果
	Signature []byte //门限签名
}

type CrossTranParam struct {
	ContractName string   //智能合约名
	ContractFunc string   //智能合约函数
	ContractArgs []string //调用参数
}

type CrossTranProof struct {
	MerklePath  [][]byte //默克尔验证路径
	TransHash   []byte   //交易hash
	Height      int
	MerkleIndex []int64
}

//节点间消息类型
type P2PMessage struct {
	Type    string
	Content string
}
