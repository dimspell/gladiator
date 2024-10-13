package backend

func (b *Backend) SendHostMigration(session *Session, isHost bool, newHostIP [4]byte) error {
	payload := []byte{}

	// Yes(int32 1)/No(int32 0)
	if isHost {
		payload = append(payload, 1, 0, 0, 0)
	} else {
		payload = append(payload, 0, 0, 0, 0)
	}

	// IP address in 4 bytes
	payload = append(payload, newHostIP[:]...)

	return b.Send(session.Conn, ChangeHost, payload)
}
