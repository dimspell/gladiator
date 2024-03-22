package proxy

import (
	"testing"
)

func TestGlobalProxy(t *testing.T) {
	// bindIP := "127.0.0.1"
	//
	// serverConn, err := net.DialTimeout("tcp", net.JoinHostPort(bindIP, "6115"), time.Second)
	// if err != nil {
	// 	t.Error(err)
	// 	return
	// }
	// defer serverConn.Close()
	//
	// hello, _ := cbor.Marshal(HelloPacket{
	// 	Game: "World",
	// 	User: "Hello",
	// })
	// hello = slices.Concat([]byte("HELLO"), hello)
	//
	// if _, err := serverConn.Write(hello); err != nil {
	// 	t.Error(err)
	// 	return
	// }
	//
	// buf := make([]byte, 100)
	// n, err := serverConn.Read(buf)
	// if err != nil {
	// 	return
	// }
	// fmt.Println(n, string(buf[:n]))
	//
	// doc, _ := cbor.Marshal(Message{
	// 	From: "hello",
	// 	To:   "World",
	// 	// Payload: nil,
	// })
	// serverConn.Write(doc)
}
