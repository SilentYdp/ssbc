package util

import (
	"fmt"
	"strconv"
	"testing"
)

func TestString(t *testing.T) {
	a:=7
	aS:=strconv.Itoa(a)
	fmt.Println(aS)
}

func TestGzip(t *testing.T)  {
	hash:="dsdadfafdff121e3edeeejfehu4ru4fneffferf"
	buff:=make([]byte,0)
	for i:=0;i<10000;i++{
		buff=append(buff,[]byte(hash)...)
	}
	fmt.Println(len(buff))
	ret:=Compress(buff)
	ori:=DeCompress(ret)
	fmt.Println(len(ori))
}
