package net

import (
	"fmt"
	"github.com/ssbc/common"
	"sync"
	"time"

	"github.com/cloudflare/cfssl/log"
)

var (
	Nodes = 4
	Urls []string = []string{
		"http://192.168.1.101:8000",
		"http://192.168.1.155:8001",
		"http://192.168.1.102:8002",
		"http://192.168.1.104:8003",
	}
	isSelfLeader bool = false  //leader
	blockState BlockState
	voteCounts int = 1
	Ports string
	Sender string = "windows"
	signatures map[string][]byte
	senders map[string]string
	transinblock int = 4000
	MaxTransInBlock int = 6000
	TransChangeStep = 200  //每一个大轮次区块中交易数量的变化幅度
	transtoredis int = 300000
	times int = 0
	rounds int = 10
	NeedZip = false   //是否需要压缩与解压缩（配套使用）
	Testflag = ""
)

type BlockState struct{
	sync.Mutex
	currBlock common.Block
	tmpBlock common.Block
}

func (bs *BlockState) GetCurrB()common.Block{
	bs.Lock()
	defer bs.Unlock()
	return bs.currBlock
}

func (bs *BlockState) GetTmpB()common.Block{
	bs.Lock()
	defer bs.Unlock()
	return bs.tmpBlock
}

func (bs *BlockState) SetCurrB(b common.Block){
	bs.Lock()
	defer bs.Unlock()
	bs.currBlock = b

}

func (bs *BlockState) SetTmpB(b common.Block){
	bs.Lock()
	defer bs.Unlock()
	bs.tmpBlock = b
}

func (bs *BlockState) Checkblock(b *common.Block)bool{
	bs.Lock()
	defer bs.Unlock()
	if b.PrevHash != bs.currBlock.Hash{
		return false
	}
	if bs.tmpBlock.Hash == b.Hash{
		return false
	}
	return true

}

func (bs *BlockState) Checks(hash string)bool{
	bs.Lock()
	defer bs.Unlock()

	if bs.tmpBlock.Hash != hash{
		return true
	}
	if bs.tmpBlock.Hash == bs.currBlock.Hash{
		return true
	}
	return false

}

func (bs *BlockState) StoreBlock(){
	bs.Lock()
	defer bs.Unlock()
	log.Info("store the block into Mysql")
	//log.Info("Successfully stored the block", bs.tmpBlock)
	common.Blockchains <- bs.tmpBlock
	bs.currBlock = bs.tmpBlock

}

func (bs *BlockState) CheckAndStore(hash string){
	bs.Lock()
	defer bs.Unlock()

	if bs.tmpBlock.Hash != hash{
		log.Info("store_block: This round may finished. not equal to hash")
		return
	}
	if bs.tmpBlock.Hash == bs.currBlock.Hash{
		log.Info("store_block: This round may finished. equal to current")
		return
	}
	log.Info("Pulling out tmpBlock")
	log.Info("store the block")
	log.Info("store the block into Mysql")
	//lib.Db.insert(block)
	//log.Info("Successfully stored the block", bs.tmpBlock)
	common.Blockchains <- bs.tmpBlock
	bs.currBlock = bs.tmpBlock
	//send block to sc
	//sendTxToSC(bs.currBlock)
	if isSelfLeader{
		//TPS估算
		t2 = time.Now()
		dura:=t2.Sub(t1).Seconds()
		log.Info("duration: ",t2.Sub(t1))
		log.Info("times and len of blockchain: ", times+1, len(common.Blockchains))
		totalTrans:=float64((times+1)*transinblock)
		tps:=totalTrans/dura
		fmt.Println("TransInBlock=",transinblock,", Rounds=",times,", TotalTrans=",totalTrans,", Duration=",dura,", TPS=",tps)
	}

	if times + 1 < rounds{
		times++
		//time.Sleep(time.Second)
		if isSelfLeader{
			go sendTrans()
		}
	}else {
		//达到了轮次要求
		times=0//重置
		flag=true //接收交易重置+起始时间重置
		transinblock=transinblock+TransChangeStep //每次加
		if isSelfLeader && transinblock<=MaxTransInBlock{
			go sendTrans()
		}
	}
}

func Init(){
	blockState.SetCurrB(common.B)
	signatures = make(map[string][]byte)
	senders = make(map[string]string)
	log.Info("Byzantium Init Successfully")
}








func vote(s *Server)*serverEndpoint{
	return &serverEndpoint{
		Methods: []string{ "POST"},
		Handler: voteHandler,
		Server:  s,
	}
}



func voteHandler(ctx *serverRequestContextImpl) (interface{}, error) {

	return nil, nil
}



func store_vote(v *Vote){

}









