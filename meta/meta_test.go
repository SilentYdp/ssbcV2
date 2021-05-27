package meta

import (
	"encoding/json"
	"fmt"
	"github.com/ssbcV2/commonconst"
	"testing"
)

func TestCrossRegister(t *testing.T) {
	node := Node{
		Id:        "QmXVPeS7f4yYmNKjmwNsy8u4WNL4fVsxUARPFBsWScr3uG",
		PublicKey: `{"N":1714287898398486379174090593007261028035824969789713135770721403572923339156391180232877898782950930817973613614092379951709185379606158142493602693530563077428945676910622416190447317874614919714568057682442060826651504829714731614744299768632702559710086068337584625145723792660542036996395145305316049,"E":65537}`,
		IP:        "127.0.0.1",
		Port:      "5505",
	}
	nodes := make([]Node, 0)
	nodes = append(nodes, node)

	info := RegisterInformation{
		Id:       "ssbc",
		Relayers: nodes,
		Servers:  nodes,
	}

	infoByte, _ := json.Marshal(info)
	log.Info(string(infoByte))
}

func TestCrossTran(t *testing.T) {
	tran := CrossTran{
		SourceChainId: "ssbc",
		DestChainId:   "ssbc2",
		Type:          commonconst.CrossTranTransferType,
		From:          "",
		To:            "",
		Value:         10,
	}
	tranByte, _ := json.Marshal(tran)
	log.Info(string(tranByte))
}
