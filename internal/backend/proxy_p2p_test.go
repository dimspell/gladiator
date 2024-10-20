package backend

// func TestPeerTopeerManual(t *testing.T) {
// 	t.Cleanup(func() {
// 		defer goleak.VerifyNone(t)
// 	})
// 	StartHost(t)

// 	const roomName = "test"
// 	const (
// 		player1Name = "player1"
// 		player2Name = "player2"
// 		player3Name = "player3"
// 		player4Name = "player4"

// 		hostRole  = Role(signalserver.RoleHost)
// 		guestRole = Role(signalserver.RoleGuest)
// 	)

// 	ws := &FakeWebsocket{Buffer: make([][]byte, 0)}

// 	player1 := &PeerToPeer{
// 		Peers:  p2p.NewPeers(),
// 		IpRing: NewIpRing(),
// 		ws:     ws,
// 	} // Host called "player1"
// 	player1HandlePackets := player1.handlePackets(hostRole, player1Name, roomName)

// 	player2 := &PeerToPeer{
// 		Peers:  p2p.NewPeers(),
// 		IpRing: NewIpRing(),
// 		ws:     ws,
// 	} // Guest, called "player2", joining to "player1"
// 	player2HandlePackets := player2.handlePackets(guestRole, player2Name, roomName)

// 	if err := player1.handleJoin(signalserver.MessageContent[signalserver.Member]{
// 		Type: signalserver.Join,
// 		Content: signalserver.Member{
// 			UserID: player2Name,
// 			Role:   signalserver.RoleGuest,
// 		},
// 		From: "",
// 		To:   "",
// 	}, player1Name, roomName, hostRole); err != nil {
// 		t.Error(err)
// 		return
// 	}

// 	player1.Peers.Range(func(_ string, peer *p2p.Peer) {
// 		<-webrtc.GatheringCompletePromise(peer.Connection)
// 	})

// 	{
// 		var arr []byte
// 		arr = make([]byte, 1024)
// 		n, err := ws.Read(arr)
// 		if err != nil {
// 			t.Error(err)
// 			return
// 		}
// 		if err := player1HandlePackets(arr[:n]); err != nil {
// 			t.Error(err)
// 			return
// 		}
// 	}

// 	if err := player2.handleJoin(signalserver.MessageContent[signalserver.Member]{
// 		Type: signalserver.Join,
// 		Content: signalserver.Member{
// 			UserID: player1Name,
// 			Role:   signalserver.RoleHost,
// 		},
// 		From: "",
// 		To:   "",
// 	}, player2Name, roomName, guestRole); err != nil {
// 		t.Error(err)
// 		return
// 	}

// 	{
// 		var arr []byte
// 		arr = make([]byte, 1024)
// 		n, err := ws.Read(arr)
// 		if err != nil {
// 			t.Error(err)
// 			return
// 		}
// 		if err := player2HandlePackets(arr[:n]); err != nil {
// 			t.Error(err)
// 			return
// 		}
// 	}

// 	player2.Peers.Range(func(_ string, peer *p2p.Peer) {
// 		<-webrtc.GatheringCompletePromise(peer.Connection)
// 	})

// 	{
// 		var arr []byte
// 		arr = make([]byte, 1024)
// 		n, err := ws.Read(arr)
// 		if err != nil {
// 			t.Error(err)
// 			return
// 		}
// 		if err := player2HandlePackets(arr[:n]); err != nil {
// 			t.Error(err)
// 			return
// 		}
// 	}

// 	{
// 		var arr []byte
// 		arr = make([]byte, 1024)
// 		n, err := ws.Read(arr)
// 		if err != nil {
// 			t.Error(err)
// 			return
// 		}
// 		if err := player1HandlePackets(arr[:n]); err != nil {
// 			t.Error(err)
// 			return
// 		}
// 	}

// 	fmt.Println(player1.Peers.Get(player2Name))
// 	fmt.Println(player2.Peers.Get(player1Name))
// 	fmt.Println("Done")

// 	player1.Close()
// 	player2.Close()

// 	// assert.NoError(t, err)

// }

// func TestPeerToPeer(t *testing.T) {
// 	defer goleak.VerifyNone(t)

// 	t.Run("Hosting a game", func(t *testing.T) {
// 		const roomName = "room"

// 		// StartHost(t)
// 		websocketURL := StartSignalServer(t)
// 		// websocketURL := "ws://localhost:5050"

// 		a := NewPeerToPeer(websocketURL)
// 		defer a.Close()

// 		if _, err := a.CreateRoom(CreateParams{
// 			HostUserIP: "",
// 			HostUserID: "user1",
// 			GameID:     roomName,
// 		}); err != nil {
// 			t.Error(err)
// 			return
// 		}
// 		if err := a.Host(HostParams{
// 			GameID:     roomName,
// 			HostUserID: "user1",
// 		}); err != nil {
// 			t.Error(err)
// 			return
// 		}

// 		b := NewPeerToPeer(websocketURL)
// 		defer b.Close()

// 		if _, err := b.Join(JoinParams{
// 			HostUserID:    "user1",
// 			GameID:        roomName,
// 			HostUserIP: "",
// 			CurrentUserID: "user2",
// 		}); err != nil {
// 			t.Error(err)
// 			return
// 		}

// 		time.Sleep(2 * time.Second)

// 		fmt.Println(a.Peers)
// 		fmt.Println(b.Peers)

// 		if _, err := b.GetPlayerAddr(GetPlayerAddrParams{
// 			GameID:    roomName,
// 			UserID:    "user1",
// 			IPAddress: "127.0.0.1",
// 		}); err != nil {
// 			t.Error(err)
// 			return
// 		}
// 	})
// }

// func StartSignalServer(t testing.TB) string {
// 	t.Helper()

// 	h, err := signalserver.NewServer()
// 	if err != nil {
// 		t.Fatal(err)
// 		return ""
// 	}
// 	ts := httptest.NewServer(h)

// 	t.Cleanup(func() {
// 		ts.Close()
// 	})

// 	wsURI, _ := url.Parse(ts.URL)
// 	wsURI.Scheme = "ws"

// 	return wsURI.String()
// }
