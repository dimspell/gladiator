package proxy

import "fmt"

func ListenTCP(i byte) {
	addr := fmt.Sprintf("127.21.37.%d:6114", i)
	fmt.Println(addr)
}

func ListenUDP(i byte) {
	addr := fmt.Sprintf("127.21.37.%d:6113", i)
	fmt.Println(addr)
}

func ReconcileConnections() {

}
