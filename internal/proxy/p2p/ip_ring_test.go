package p2p_test

import (
	"fmt"

	"github.com/dimspell/gladiator/internal/proxy/p2p"
)

func ExampleIpRing_NextAddr() {
	r := p2p.NewIpRing()
	r.IsTesting = true
	r.TcpPortPrefix = 1234

	fmt.Println(r.NextAddr())
	fmt.Println(r.NextAddr())
	fmt.Println(r.NextAddr())
	fmt.Println(r.NextAddr())
	fmt.Println(r.NextAddr())
	// Output:
	// 127.0.0.1 12342 61132
	// 127.0.0.1 12343 61133
	// 127.0.0.1 12344 61134
	// 127.0.0.1 12342 61132
	// 127.0.0.1 12343 61133
}
