package net

import (
	"github.com/cloudflare/cfssl/log"
	"github.com/ssbc/common"
	"encoding/json"
	"github.com/ssbc/lib/redis"
	rd "github.com/gomodule/redigo/redis"
	"github.com/ssbc/util"
	"time"
)

func receiveBlock(s *Server)*serverEndpoint{
	return &serverEndpoint{
		Methods: []string{ "POST"},
		Handler: receiveBlockHandler,
		Server:  s,
	}
}

func receiveBlockHandler(ctx *serverRequestContextImpl) (interface{}, error) {
	b:=make([]byte,0)
	if NeedZip{
		bzip,err := ctx.ReadBodyBytes()
		if err !=nil{
			log.Info("ERR receiveBlockHandler: ", err)
		}
		//解压缩
		b=util.DeCompress(bzip)
	}else {
		b,_=ctx.ReadBodyBytes()
	}

	//log.Info("receiveNewBlock")
	bs:=common.BlockMsg{}
	json.Unmarshal(b,&bs)
	//先进行验签
	data,_:=json.Marshal(bs.Bc)
	if !util.RsaVerySignWithSha256(data,bs.Sign,bs.Pk){
		log.Info("[receiveBlockHandler] 验签不通过")
		return nil, nil
	}

	//newBlock := &common.Block{}
	//err := json.Unmarshal(b, newBlock)
	//if err !=nil{
	//	log.Info("ERR receiveBlockHandler: ", err)
	//}
	newBlock:=&bs.Bc
	log.Info("receiveBlockHandler newBlock")
	if !blockState.Checkblock(newBlock){
		log.Info("receiveBlockHandler: Hash mismatch. This round may finish")
		return nil, nil
	}
	switch Testflag {
	case "rtb":
		t2 = time.Now()
		dura := t2.Sub(t1)
		log.Info("duration: ",dura)
		log.Info("times : ", times+1 )
		if times + 1 < rounds{
			times++
			//time.Sleep(time.Second)
			SendTransV2()
		}
		return nil, nil
	}
	go verify(newBlock)
	return nil, nil
}

func verify(block *common.Block){
	// sender
	//tmpblock change to cache to redis next version
	voteBool := false
	if verify_block(block){
		voteBool = true
	}
	log.Info("verify block: ",voteBool)
	blockState.SetTmpB(*block)
	v := &Vote{Sender : Sender, Hash : block.Hash, Vote : voteBool }
	b, err := json.Marshal(v)
	if err != nil{
		log.Info("verify_block: ", err)
	}
	log.Info("vote: ", string(b))

	//先进行签名
	vs:=VoteMsg{}
	vs.Pk=common.PublicKey
	vs.Vt=*v
	vs.Sign=util.RsaSignWithSha256(b,common.PrivateKey)
	vsB,_:=json.Marshal(vs)
	Broadcast("recBlockVoteRound1", vsB)
}


func verify_block(block *common.Block)bool{
	//验证逻辑 验签 验证交易 验证merkle tree root
	currentBlock := blockState.GetCurrB()
	if block.PrevHash != currentBlock.Hash{
		log.Info("This round may finish")
		return false
	}
	if block.Signature != "Signature"{
		log.Info("verify block: Signature mismatch")
		return false
	}
	return verifyBlockTxV2(block,&currentBlock)
}

func verifyBlockTxV2(b *common.Block, currentBlock *common.Block)bool{
	return true
	if len(b.TX)!=len(commonTransList){
		log.Info("len b.Tx=",len(b.TX),",lencommonTransList=",len(commonTransList))
		return false
	}
	return true
}

func verifyBlockTx(b *common.Block, currentBlock *common.Block)bool{
	return true
	transCache := []interface{}{"verifyBlockTxCache"+b.Hash}
	for _,data := range b.TX{
		b,err := json.Marshal(data)
		if err != nil{
			log.Info("verifyBlockTx json err: ", err)
		}
		transCache = append(transCache, b)
	}
	if len(transCache) == 1{
		transCache = append(transCache, []byte{})
	}
	conn := redis.Pool.Get()
	defer conn.Close()
	_,err := conn.Do("SADD", transCache...)
	if err != nil{
		log.Info("verifyBlockTx err SADD: ", err)
	}

	//使用的是用一个redis，在分布式的场景中无法如此判别
	//commonTrans,err := rd.Strings(conn.Do("SINTER", "verifyBlockTxCache"+b.Hash, "CommonTxCache4verify"+ currentBlock.Hash))
	//commonTrans,err := rd.Strings(conn.Do("SMEMBERS", "verifyBlockTxCache"+b.Hash))
	commonTrans,err := rd.Strings(conn.Do("SINTER", "verifyBlockTxCache"+b.Hash, "verifyBlockTxCache"+b.Hash))


	if err !=nil{
		log.Info("verifyBlockTx err SINTER: ", err)
	}
	log.Info("verifyBlockTx commonTrans:   ", commonTrans)
	log.Info("verifyBlockTx len trans commonTrans :   ", len(b.TX), len(commonTrans))
	if len(b.TX) != len(commonTrans){
		return false
	}
	return true
}
