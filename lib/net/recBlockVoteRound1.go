package net

import (
	"encoding/json"
	"github.com/cloudflare/cfssl/log"
	rd "github.com/gomodule/redigo/redis"
	"github.com/ssbc/lib/redis"
	"strconv"
)

type Vote struct{
	Sender string
	Hash string
	Vote bool
}

func recBlockVoteRound1(s *Server)*serverEndpoint{
	return &serverEndpoint{
		Methods: []string{ "POST"},
		Handler: recBlockVoteRound1Handler,
		Server:  s,
	}
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
		go voteForRoundNew(v.Hash,ctx.req.Host+timesS+tIBS)
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
