package backend

// func sendHostMigration(conn net.Conn, successful bool, newHostIP [4]byte) error {
// 	packet := []byte{255, opChangeHost, 0, 0}
//
// 	// Have the host changed? Yes(int32 1)/No(int32 0)
// 	if successful {
// 		packet = append(packet, 1, 0, 0, 0)
// 	} else {
// 		packet = append(packet, 0, 0, 0, 0)
// 	}
//
// 	// IP address in 4 bytes
// 	// packet = append(packet, 127, 0, 0, 1)
// 	packet = append(packet, newHostIP[:]...)
//
// 	binary.LittleEndian.PutUint16(packet[2:4], uint16(len(packet)))
// 	_, err := conn.Write(packet)
// 	return err
// }
