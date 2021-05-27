package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestGet(t *testing.T)  {
	a:=make(map[string]string)
	a["height"]="0"
	a["dest"]="ssbc2"


	r:=ContractRequest{
		Method: "GetAbstractBlockHeader",
		Args:   a,
	}

	rb,_:=json.Marshal(r)
	log.Info(string(rb))
}
