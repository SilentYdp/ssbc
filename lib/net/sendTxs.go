package net

import (
	"github.com/cloudflare/cfssl/log"
)

func sendTxs(s *Server) *serverEndpoint {
	return &serverEndpoint{
		Methods: []string{"GET", "POST", "HEAD"},
		Handler: sendTxHandler,
		Server:  s,
		successRC: 200,
	}
}

//每个节点将自己redis中的交易广播给其他的节点
func sendTxHandler(ctx *serverRequestContextImpl) (interface{}, error) {
	log.Info("Send Trans start!")
	go sendTrans()
	resp := TestInfoResponseNet{
		TName: "start",
		Version: "TPS test",
	}
	return resp, nil
}

func sendTrans() {
	Broadcast("testinfo",nil)
}




