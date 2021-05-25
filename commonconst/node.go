package commonconst

//客户端与节点通信的监听地址
const ClientToNodeAddr = "127.0.0.1:8888"

//客户端与前端用户通信监听地址
const ClientToUserAddr = ":9999"

//节点个数
const NodeCount = 4

//生成全局变量-节点池
var NodeTable map[string]string

func init() {
	NodeTable = make(map[string]string)
	NodeTable = map[string]string{
		"N0":     "127.0.0.1:8000",
		"N1":     "127.0.0.1:8001",
		"N2":     "127.0.0.1:8002",
		"N3":     "127.0.0.1:8003",
	}
}
