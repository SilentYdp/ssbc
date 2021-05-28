package net

import (
	"github.com/cloudflare/cfssl/log"
	"encoding/json"
	"github.com/ssbc/lib/redis"
	rd "github.com/gomodule/redigo/redis"
	"strconv"
)

type ReVote struct{
	Sender string
	Vote []Vote
	Hash string
	V bool
}

func recBlockVoteRound2(s *Server)*serverEndpoint{
	return &serverEndpoint{
		Methods: []string{ "POST"},
		Handler: recBlockVoteRound2Handler,
		Server:  s,
	}
}

func recBlockVoteRound2Handler(ctx *serverRequestContextImpl) (interface{}, error) {
	b,err := ctx.ReadBodyBytes()
	if err !=nil{
		log.Info("ERR recBlockVoteRound2Handler: ", err)
	}
	//log.Info("recBlockVoteRound2Handler: ",string(b))
	v := &ReVote{}
	err = json.Unmarshal(b, v)
	if err !=nil{
		log.Info("ERR recBlockVoteRound2Handler: ", err)
	}
	conn := redis.Pool.Get()
	defer conn.Close()

	log.Info("第二轮投票")
	//_,err = conn.Do("SADD",v.Hash+"round2"+ctx.req.Host, ctx.req.RemoteAddr)
	//if err != nil{
	//	log.Info("recBlockVoteRound2Handler err SADD: ", err)
	//}
	//vc,err := redis.ToInt(conn.Do("SCARD",v.Hash+"round2"+ctx.req.Host))
	//if err !=nil{
	//	log.Info("recBlockVoteRound2Handler err:", err)
	//}
	timesS:=strconv.Itoa(times)
	tIBS:=strconv.Itoa(transinblock)
	_,err = conn.Do("LPUSH", v.Hash+"round2"+ctx.req.Host+timesS+tIBS, b)
	if err != nil{
		log.Info("recBlockVoteRound2Handler err LPUSH: ", err)
	}
	vc,err:=redis.ToInt(conn.Do("LLEN", v.Hash+"round2"+ctx.req.Host+timesS+tIBS))
	if err!=nil{
		log.Info("recBlockVoteRound2Handler err LLEN:",err)
	}
	log.Info("recBlockVoteRound2Handler revoteCount : ",vc)
	if vc == Nodes{
		log.Info("statistic the votes")
		go statistic(v.Hash,ctx.req.Host+timesS+tIBS)
	}
	return nil, nil
}

func statistic(hash string,host string){
	//statistic 2 round vote
	// then decide whether store the block or not
	if blockState.Checks(hash){
		log.Info("store_block: This round may finished")
		return
	}
	conn := redis.Pool.Get()
	defer conn.Close()

	//vs,err := rd.ByteSlices(conn.Do("SMEMBERS", hash+"round2"))
	//if err != nil{
	//	log.Info("recBlockVoteRound2Handler err SMEMBERS: ", err)
	//}

	vs,err := rd.ByteSlices(conn.Do("LRANGE", hash+"round2"+host,0,-1))
	if err != nil{
		log.Info("recBlockVoteRound2Handler err LRANGE: ", err)
	}
	revotes := []ReVote{}
	for _,data := range vs{
		t := ReVote{}
		err := json.Unmarshal(data, &t)
		if err != nil{
			log.Info("recBlockVoteRound2Handler err json: ", err)
			continue
		}
		revotes = append(revotes, t)
	}
	votecount := 0
	for _,data := range revotes{
		if data.V{
			votecount++
		}
	}
	log.Info("revote for ture: ", votecount)
	if float64(votecount) > float64(Nodes)* 0.75{
		log.Info("recBlockVoteRound2Handler: vote round tow has received more than 2/3 affirmative votes")
		store_block(hash)
	}
}


func store_block(hash string){
	blockState.CheckAndStore(hash)


}
