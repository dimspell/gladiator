package p2p

import (
	"fmt"
)

func ExampleIpRing_NextAddr() {
	r := NewIpRing()
	r.IsTesting = true
	r.TcpPortPrefix = 1234

	fmt.Println(r.NextAddr())
	fmt.Println(r.NextAddr())
	fmt.Println(r.NextAddr())
	fmt.Println(r.NextAddr())
	fmt.Println(r.NextAddr())
	// Output:
	// 127.0.0.1 12342 61132 <nil>
	// 127.0.0.1 12343 61133 <nil>
	// 127.0.0.1 12344 61134 <nil>
	// 127.0.0.1 12342 61132 <nil>
	// 127.0.0.1 12343 61133 <nil>
}
