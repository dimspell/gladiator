package packetlogger

import (
	"container/ring"
	"fmt"
	"testing"
)

var r = ring.New(3)

func TestName(t *testing.T) {
	n := r.Len()
	for i := 0; i < n; i++ {
		r.Value = i + 2
		r = r.Next()
	}

	r.Do(func(a any) {
		fmt.Println(a)
	})

	// fmt.Println(r.Value)
	// fmt.Println(r.Next().Value)
}
