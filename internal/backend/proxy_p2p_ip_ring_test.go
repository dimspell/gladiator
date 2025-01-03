package backend_test

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
	// 127.0.0.1 12342 61132
	// 127.0.0.1 12343 61133
	// 127.0.0.1 12344 61134
	// 127.0.0.1 12342 61132
	// 127.0.0.1 12343 61133
}

// func TestRedirects(t *testing.T) {
// 	// User 1 - Host
// 	// - User2 - OtherPlayerHasJoined
// 	// User 2 - Client
// 	// - User1 - OtherPlayerIsHost
//
// 	helperStartGameServer(t)
//
// 	r1 := NewIpRing()
// 	r1.IsTesting = true
// 	r1.UdpPortPrefix = 1300
// 	r1.TcpPortPrefix = 1400
//
// 	r2 := NewIpRing()
// 	r2.IsTesting = true
// 	r2.UdpPortPrefix = 2300
// 	r2.TcpPortPrefix = 2400
//
// 	host := r1.NextPeerAddress("2", true, true)
// 	hostTcp, hostUdp, err := redirect.New(host.Mode, host.Addr)
// 	if err != nil {
// 		t.Fatal(err)
// 		return
// 	}
// 	fmt.Println(hostTcp, hostUdp)
//
// 	// select {}
// 	// guest := r2.NextPeerAddress("1", false, true)
// }
