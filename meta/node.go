package meta

//节点
type Node struct {
	Id            string //节点Id
	PublicKey     []byte //节点公钥
	PrivateKey    []byte //节点私钥
	Adress        string //节点地址
	IsPrimaryNode bool   //是否是主节点
	IsRelayNode   bool   //是否是中继节点
	IsServerNode  bool   //是否是服务节点
}
