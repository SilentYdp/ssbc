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
