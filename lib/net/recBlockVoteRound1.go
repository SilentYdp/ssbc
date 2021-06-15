package net

import (
	"encoding/json"
	"github.com/cloudflare/cfssl/log"
	rd "github.com/gomodule/redigo/redis"
	"github.com/ssbc/common"
	"github.com/ssbc/lib/redis"
	"github.com/ssbc/util"
	"strconv"
	"sync"
)

var round1Map sync.Map


type Vote struct{
	Sender string
	Hash string
	Vote bool
}

type VoteMsg struct {
	Vt  Vote
	Sign []byte
	Pk   []byte
}

func recBlockVoteRound1(s *Server)*serverEndpoint{
	return &serverEndpoint{
		Methods: []string{ "POST"},
		Handler: recBlockVoteRound1HandlerV2,
		Server:  s,
	}
}
func vote1(b []byte)  {

	//log.Info("recBlockVoteRound1Handler: ",string(b))
	//先进行验签
	vs:=VoteMsg{}
	json.Unmarshal(b,&vs)
	data,_:=json.Marshal(vs.Vt)
	if !util.RsaVerySignWithSha256(data,vs.Sign,vs.Pk){
		log.Info("[recBlockVoteRound1HandlerV2] 验签不通过")
		return
	}

	//v := &Vote{}
	//err = json.Unmarshal(b, v)
	//if err !=nil{
	//	log.Info("ERR recBlockVoteRound1Handler: ", err)
	//}
	v:=&vs.Vt
	log.Info("第一轮投票：",*v)
	timesS:=strconv.Itoa(times)
	tIBS:=strconv.Itoa(transinblock)
	key := timesS + "_" +"_"+tIBS+"_"+ "round1_" + v.Hash
	if val,ok:=round1Map.Load(key);ok{
		votes:=val.([]Vote)
		votes=append(votes,*v)
		round1Map.Store(key,votes)
		val,_=round1Map.Load(key)
		votes=val.([]Vote)
		if len(votes)==Nodes{
			//说明广播已收齐
			log.Info("Received all votes,start round2")
			//投票鉴别是否投同意
			count := 0
			for _, v := range votes {
				if v.Vote {
					//同意票+1
					count++
				}
			}

			re := false
			if float64(count) > float64(Nodes)*0.75 {
				re = true
			}
			//开始第二轮投票
			rv := ReVote{
				Sender: Sender,
				Vote:   votes,
				Hash:   v.Hash,
				V:      re,
			}
			rvB, _ := json.Marshal(rv)
			log.Info("recBlockVoteRound1Handler vote: ", string(rvB))
			//要先签名
			rvMsg:=ReVoteMsg{}
			rvMsg.Pk=common.PublicKey
			rvMsg.RV=rv
			rvMsg.Sign=util.RsaSignWithSha256(rvB,common.PrivateKey)

			rvMsgB,_:=json.Marshal(rvMsg)

			Broadcast("recBlockVoteRound2", rvMsgB)
		}else {
			//说明广播还没收齐
			log.Info("Round1 Not receive all votes:", votes)
			return
		}
	}else {
		votes:=make([]Vote,0)
		votes=append(votes,*v)
		round1Map.Store(key,votes)
		return
	}
	return
}

func recBlockVoteRound1HandlerV2(ctx *serverRequestContextImpl) (interface{}, error) {
	//第一轮投票使用内存取代redis统计
	//为了防止并发读写map导致错乱，使用协程安全的sync.Map
	//第一轮投票使用内存取代redis统计
	//为了防止并发读写map导致错乱，使用协程安全的sync.Map
	b,err := ctx.ReadBodyBytes()
	if err !=nil{
		log.Info("ERR recBlockVoteRound1Handler: ", err)
		return nil, nil
	}
	go vote1(b)
	return nil, nil
}

func recBlockVoteRound1Handler(ctx *serverRequestContextImpl) (interface{}, error) {
	b,err := ctx.ReadBodyBytes()
	if err !=nil{
		log.Info("ERR recBlockVoteRound1Handler: ", err)
	}
	//log.Info("recBlockVoteRound1Handler: ",string(b))
	v := &Vote{}
	err = json.Unmarshal(b, v)
	if err !=nil{
		log.Info("ERR recBlockVoteRound1Handler: ", err)
	}
	log.Info("第一轮投票：",*v)
	//log.Info("ctx.host=：",ctx.req.Host,"ctx.remote=",ctx.req.RemoteAddr)
	conn := redis.Pool.Get()
	defer conn.Close()

	//_,err = conn.Do("SADD", v.Hash+"round1"+ctx.req.Host, ctx.req.RemoteAddr)
	//if err != nil{
	//	log.Info("recBlockVoteRound1Handler err SADD: ", err)
	//}
	//因为b都一样，所以这个地方得改写成允许重复元素的插入
	timesS:=strconv.Itoa(times)
	tIBS:=strconv.Itoa(transinblock)
	_,err = conn.Do("LPUSH", v.Hash+"round1"+ctx.req.Host+timesS+tIBS, b)
	if err != nil{
		log.Info("recBlockVoteRound1Handler err LPUSH: ", err)
	}
	vc,err:=redis.ToInt(conn.Do("LLEN", v.Hash+"round1"+ctx.req.Host+timesS+tIBS))
	if err!=nil{
		log.Info("recBlockVoteRound1Handler err LLEN:",err)
	}
	//每次统计之前睡一秒，给别人一个机会塞进去--这个地方好像不用睡觉了
	//time.Sleep(1*time.Second)
	//vc,err := redis.ToInt(conn.Do("SCARD",v.Hash+"round1"+ctx.req.Host))
	//if err !=nil{
	//	log.Info("recBlockVoteRound1Handler err SCARD:", err)
	//}
	log.Info("recBlockVoteRound1Handler voteCount : ",vc)
	if vc == Nodes{
		log.Info("voteForRoundTwo")
		voteForRoundNew(v.Hash,ctx.req.Host+timesS+tIBS)
	}
	return nil, nil
}

func voteForRoundNew(hash string,host string){
	//when receive whole nodes votes
	//then statistics
	conn := redis.Pool.Get()
	defer conn.Close()

	votes,err := rd.ByteSlices(conn.Do("LRANGE", hash+"round1"+host,0,-1))
	if err != nil{
		log.Info("recBlockVoteRound1Handler err LRANGE: ", err)
	}
	vs:=[]Vote{}
	for _,data := range votes{
		t := Vote{}
		err := json.Unmarshal(data, &t)
		if err != nil{
			log.Info("recBlockVoteRound1Handler err Json:", err)
			continue
		}
		vs = append(vs, t)
	}
	votecount := 0
	for _,v := range vs{
		if v.Vote{
			votecount++
		}
	}
	v := false
	if float64(votecount) > float64(Nodes)*0.75{
		log.Info("voteForRoundTwo: vote round has received more the 3/4 affirmative vote")
		v = true
	}
	rv := &ReVote{Sender: Sender, Vote: vs, Hash: hash, V: v}
	b, err := json.Marshal(rv)
	if err != nil{
		log.Info("voteForRoundTwo err json: ",err)
	}
	log.Info("recBlockVoteRound1Handler vote: ", string(b))
	Broadcast("recBlockVoteRound2",b)

}
