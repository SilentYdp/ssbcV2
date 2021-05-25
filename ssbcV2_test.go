package main

import (
	"github.com/ssbcV2/smart_contract"
	"testing"
)

func TestBuildAndRun(t *testing.T) {
	smart_contract.BuildAndRun("./smart_contract/test/","test")
}
