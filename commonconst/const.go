package commonconst

//锁定账户
const LockedAccount = "lockedAccount"

//跨链交易类型
const CrossTranReadType = "CTR"     //跨链读
const CrossTranWriteType = "CTW"    //跨链写
const CrossTranTransferType = "CTT" //跨链转账

//跨链交易状态类型
const StatusDeal = "deal"
const StatusSuccess = "success"
const StatusFailed = "failed"

//p2p消息类型
const BlockChainSynchronizeMsg = "BCS"
const AbstractBlockHeaderSynchronizeMsg = "ABHS"
const VRFMsg = "VRFM"
const VRFOrderMsg = "VRFO"
const AbstractHeaderProposalMsg = "AbstractHeaderProposal"
const TssMsg = "tssmsg" //门限签名消息
const HeaderProposalSignMsg = "HeaderProposalSignMsg"

//TCP消息类型
const TcpPing = "ping"
const TcpPong = "pong"
const TcpAbstractHeader = "abstract_header"
const TcpCrossTrans = "tcp_cross_trans"
const TcpCrossTransReceipt = "tcp_cross_trans_receipt"

//redis key
const BlockChainKey = "BlockChain"
const TransActionsKey = "transactions"
const AbstractHeadersKey = "abstractHeaders"
const AbstractHeadersFinalKey = "abstractHeadersFinal"
const RegisterInformationKey = "register_information"
const RemoteHeadersKey = "remote_headers_key"

//客户端命令
const TXOrder = "tx"
const VrfOrder = "vrf"
const RegisterOrder = "register"
const TCPConnectOrder = "conn"                  //TCP握手
const CrossTransOrder = "cto"                   //发送跨链交易
const RemoteChainHeaderSynchronizeOrder = "rcs" //给他链发送区块头同步信息

//交易数阈值
const TxsThreshold = 4

//恶意节点数阈值
const EvilCount = 1

const ProposalSignCount = 1
const VRFThreshold = 1

const LocalChainId = "ssbc"
const LocalChainId2 = "ssbc2"
