package net

import (
	"encoding/json"
	"github.com/cloudflare/cfssl/log"
	"github.com/deckarep/golang-set"
	rd "github.com/gomodule/redigo/redis"
	"github.com/ssbc/common"
	"github.com/ssbc/lib/redis"
	"github.com/ssbc/util"
	"strconv"
	"sync"
)


var commonTransMap sync.Map
var commonTransList []common.Transaction

type TransHash struct {
	BlockHash string
	TransHashs []string
	Pk         []byte
	Sign       []byte
}

func receive_trans_bitarry(s *Server)*serverEndpoint{
	return &serverEndpoint{
		Methods: []string{ "POST"},
		Handler: rtbitarryHandler,
		Server:  s,
	}
}

func receiveTran(b []byte)  {
	transHash := TransHash{}
	err := json.Unmarshal(b, &transHash)
	if err != nil{
		log.Info("rtbitarry json err: ", err)
		return
	}

	//先进行验签，验签不通过直接退出
	data,_:=json.Marshal(transHash.TransHashs)
	if !util.RsaVerySignWithSha256(data,transHash.Sign,transHash.Pk){
		log.Info("[rtbitarryHandler] 验签不通过")
		return
	}
	//log.Info("rtbitarryHandler receiving: ", string(b))
	//判断自己是否收到了四份交易，收到四份之后才能进行公共交易集的筛选
	//log.Info("接收交易：",transHash)
	timesS:=strconv.Itoa(times)
	tIBS:=strconv.Itoa(transinblock)
	key := timesS + "_" +"_"+tIBS+"_"+ "commonTrans"
	if val,ok:=commonTransMap.Load(key);ok{
		ts:=val.([]TransHash)
		ts=append(ts,transHash)
		commonTransMap.Store(key,ts)
		val,_=commonTransMap.Load(key)
		ts=val.([]TransHash)
		if len(ts)==Nodes{
			//说明广播已收齐
			log.Info("Received all trans,start find common trans")
			//使用性能强悍的golang-set寻找公共集
			commonTranHashs := mapset.NewSet()
			for _, tr := range ts {
				for _, h := range tr.TransHashs {
					commonTranHashs.Add(h)
				}
			}
			commonTranHashsSlice := commonTranHashs.ToSlice()
			currentBlock := blockState.GetCurrB()
			if transHash.BlockHash != currentBlock.Hash{
				log.Info("findCommonTrans: BlockHash mismatch. This round may finish.")
				return
			}
			go generateFromCommonTxV2(commonTranHashsSlice,currentBlock)
		}else {
			//说明广播还没收齐
			log.Info("Not receive all trans")
			return
		}
	}else {
		ts:=make([]TransHash,0)
		ts=append(ts,transHash)
		commonTransMap.Store(key,ts)
		return
	}
	return
}

//接收交易hash
func rtbitarryHandler(ctx *serverRequestContextImpl) (interface{}, error) {
	b:=make([]byte,0)
	if NeedZip{
		bzip,err := ctx.ReadBodyBytes()
		if err != nil{
			log.Info("rtbitarry Readbody err: ", err)
			return nil,nil
		}
		//解压缩
		b=util.DeCompress(bzip)
	}else {
		b,_=ctx.ReadBodyBytes()
	}
	go receiveTran(b)
	return nil, nil
}

func findCommonTransV2(trans TransHash, sender string)  {

	//if !isSelfLeader {
	//	log.Info("findCommonTrans: Not Leader", isSelfLeader)
	//	return
	//}
	//log.Info("Leader Mode")
	currentBlock := blockState.GetCurrB()
	if trans.BlockHash != currentBlock.Hash{
		log.Info("findCommonTrans: BlockHash mismatch. This round may finish.")
		return
	}
	//generateFromCommonTxV2(trans.TransHashs, currentBlock)
}

func generateFromCommonTxV2(commonTrans []interface{}, currentBlock common.Block)  {
	//基于redis存储hash与对象的映射关系，再在for循环中进行反序列化太耗时了
	//所以优化为在内存中存储hash与对象的映射关系，一个区块中顶多几千笔交易，这种内存占用是不会造成内存打满的
	trans := []common.Transaction{}
	for _,t:=range commonTrans{
		if tr,ok:=HashTranMap[t.(string)];ok{
			trans=append(trans,tr)
		}
	}
	//为后面的公共集验证做准备
	commonTransList=trans
	//只有主节点才有资格进行建块广播
	if isSelfLeader{
		b := common.Block{TX:trans}
		b = common.GenerateBlock(currentBlock, b)
		bb, err := json.Marshal(b)
		if err != nil{
			log.Info("generateFromCommonTx err json block: ", err)
			return
		}
		bs:=common.BlockMsg{}
		//进行签名
		bs.Bc=b
		bs.Sign=util.RsaSignWithSha256(bb,common.PrivateKey)
		bs.Pk=common.PublicKey

		bsB,_:=json.Marshal(bs)

		if NeedZip{
			bszip:=util.Compress(bsB)
			Broadcast("recBlock",bszip)
		}else {
			Broadcast("recBlock",bsB)
		}
	}
}


func findCommonTrans(trans TransHash, sender string){

	currentBlock := blockState.GetCurrB()
	if trans.BlockHash != currentBlock.Hash{
		log.Info("findCommonTrans: BlockHash mismatch. This round may finish.")
		return
	}
	conn := redis.Pool.Get()
	defer conn.Close()
	data := []interface{}{"CommonTx"+ currentBlock.Hash + sender}
	for _,d := range trans.TransHashs{
		data = append(data, d)
	}
	//redis the trans
	_,err := conn.Do("SADD", data...)
	if err != nil{
		log.Info("findCommonTrans err SADD trans: ", err)
	}
	//redis the senders
	_,err = conn.Do("SADD", "CommonTx"+ currentBlock.Hash, "CommonTx"+ currentBlock.Hash+ sender)
	if err != nil{
		log.Info("findCommonTrans err SADD senders: ", err)
	}
	//check the len of the senders

	l,err := rd.Int(conn.Do("SCARD", "CommonTx"+ currentBlock.Hash))
	if err !=nil{
		log.Info("findCommonTrans err SCARD: ", err)
	}
	//Leader mode and check if got enough nodes tranx
	if !isSelfLeader {
		log.Info("findCommonTrans: Not Leader", isSelfLeader)
		return
	}
	log.Info("Leader Mode")
	//这个地方没办法，不可能通过redis来通信，肯定得让主节点知道其他的节点都收到了交易
	l=Nodes
	if l != Nodes{
		log.Info("findCommonTrans: Do not get enough nodes ", l)
		return
	}
	//find the common trans
	senders,err := rd.Strings(conn.Do("SMEMBERS", "CommonTx"+ currentBlock.Hash))
	if err !=nil{
		log.Info("findCommonTrans err SMEMBERS: ", err)
	}
	senderInterface := []interface{}{}
	for _,s := range senders{
		senderInterface = append(senderInterface, s)
	}
	//注释
	//log.Info("senderInterface: ", senderInterface)
	commonTrans,err := rd.Strings(conn.Do("SINTER", senderInterface...))
	if err !=nil{
		log.Info("findCommonTrans err SINTER: ", err)
	}
	//log.Info("findCommonTrans commonTrans: ", commonTrans)
	generateFromCommonTx(commonTrans, currentBlock)
}

func generateFromCommonTx(commonTrans []string, currentBlock common.Block){
	conn := redis.Pool.Get()
	defer conn.Close()
	mb,err := rd.Bytes(conn.Do("GET", "CommonTxCache"+ currentBlock.Hash))
	if err !=nil{
		log.Info("generateFromCommonTx err GET: ", err)
	}
	m := make(map[string][]byte)
	err = json.Unmarshal(mb, &m)
	if err !=nil{
		log.Info("generateFromCommonTx err GET: ", err)
	}
	trans := []common.Transaction{}
	for _,s := range commonTrans{
		if v,ok := m[s]; ok{
			tx := common.Transaction{}
			err := json.Unmarshal(v, &tx)
			if err != nil{
				log.Info("generateFromCommonTx err json trans: ", err)
				continue
			}
			trans = append(trans, tx)
		}
	}
	b := common.Block{TX:trans}
	b = common.GenerateBlock(currentBlock, b)
	bb, err := json.Marshal(b)
	if err != nil{
		log.Info("generateFromCommonTx err json block: ", err)
		return
	}

	//先压缩
	bbzip:=util.Compress(bb)
	Broadcast("recBlock",bbzip)
}


