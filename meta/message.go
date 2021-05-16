package meta

//tcp传递消息
type TCPMessage struct {
	Type    string //消息类型
	Content []byte //消息体
	From    string
	To      string
}
