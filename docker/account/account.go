package account

/*
账户管理智能合约
主要函数：
注册账户

*/
type AccountManager struct {
}

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


func (a *AccountManager)Register() {

}
