package meta

//账户

type Account struct {
	Address    string //账户地址
	Balance    int    //账户余额
	Data       AccountData
	PublicKey  []byte //账户公钥
	PrivateKey []byte //账户私钥
}

type AccountData struct {
	Code         string //合约代码
	ContractName string //合约名称
}
