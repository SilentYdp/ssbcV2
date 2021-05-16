package main

import "testing"

func TestGetBlockChain(t *testing.T) {
	s := NewClientServer(clientHttpAddr)
	s.Start()
}
