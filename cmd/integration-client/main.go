// Command integration-client is the mock Gladiator game client used by the
// multi-docker integration tests. It is copied into the test image
// (Dockerfile.integration) and runs inside the backend container's network
// namespace so it can reach loopback fake hosts (127.0.0.X) created by the
// relay/p2p proxies.
//
// This is the SPIKE stub: it validates the container topology by connecting to
// the backend and binding a loopback UDP socket.
package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	backendAddr := os.Getenv("BACKEND_ADDR")
	if backendAddr == "" {
		backendAddr = "127.0.0.1:6112"
	}

	// 1) Prove the backend is reachable in the shared network namespace.
	conn, err := net.DialTimeout("tcp", backendAddr, 5*time.Second)
	if err != nil {
		fmt.Printf("SPIKE_FAIL: cannot reach backend %s: %v\n", backendAddr, err)
		os.Exit(1)
	}
	_ = conn.Close()
	fmt.Printf("SPIKE_OK: reached backend %s\n", backendAddr)

	// 2) Prove loopback aliases (127.0.0.X) are usable in this namespace — this
	// is what relay/p2p fake hosts bind to.
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.2:6113")
	if err != nil {
		fmt.Printf("SPIKE_FAIL: resolve loopback: %v\n", err)
		os.Exit(1)
	}
	pc, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Printf("SPIKE_FAIL: cannot bind 127.0.0.2:6113: %v\n", err)
		os.Exit(1)
	}
	_ = pc.Close()
	fmt.Println("SPIKE_OK: loopback 127.0.0.2:6113 bindable")

	fmt.Println("INTEGRATION_OK")
}
