package client

// type DialParams struct {
// 	SignalingURL string
// 	RoomName     string
// 	ID           string
// 	Name         string
// }
//
// func Dial(params *DialParams) (*Client, error) {
// 	// Parse the signaling URL provided from the parameters (command flags)
// 	u, err := url.Parse(params.SignalingURL)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	// Set parameters
// 	v := u.Query()
// 	v.Set("userID", params.ID)
// 	v.Set("roomName", params.RoomName)
// 	u.RawQuery = v.Encode()
//
// 	slog.Debug("Connecting to the signaling server", "url", u.String())
// 	ws, err := websocket.Dial(u.String(), "", "http://localhost:8080")
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	// Send "hello" message to the signaling server
// 	req := &signalserver.Message{
// 		From:    params.ID,
// 		Type:    signalserver.HandshakeRequest,
// 		Content: params.Name,
// 	}
// 	if _, err := ws.Write(req.ToCBOR()); err != nil {
// 		return nil, err
// 	}
//
// 	// Read the response from the signaling server
// 	buf := make([]byte, 128)
// 	n, err := ws.Read(buf)
// 	if err != nil {
// 		slog.Error("Error reading message", "error", err)
// 		return nil, err
// 	}
// 	if n == 0 || buf[0] != byte(signalserver.HandshakeResponse) {
// 		return nil, fmt.Errorf("unexpected handshake response: %v", buf[:n])
// 	}
// 	resp, err := decodeCBOR[signalserver.MessageContent[string]](buf[1:n])
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	slog.Info("Connected to signaling server", "response", resp.Content)
//
// 	return &Client{
// 		ID:   params.ID,
// 		Name: params.Name,
//
// 		Peers: NewPeers(),
// 		ws:    ws,
// 	}, nil
// }
//
// func (c *Client) Close() {
// 	if c == nil {
// 		return
// 	}
// 	if c.ws != nil {
// 		if err := c.ws.Close(); err != nil {
// 			slog.Warn("Could not close websocket connection", "error", err)
// 		}
// 	}
// }
//
// type Client struct {
// 	ID   string
// 	Name string
//
// 	// ws is the websocket connection to the signaling server
// 	ws *websocket.Conn
//
// 	// peers stores the WebRTC peer connections
// 	Peers *Peers
// }
//
// type MessageHandler func(peer *Peer, packet webrtc.DataChannelMessage)
//
// func (c *Client) Run(onUDP MessageHandler, onTCP MessageHandler) {
// 	for {
// 		buf := make([]byte, 1024)
// 		n, err := c.ws.Read(buf)
// 		if err != nil {
// 			log.Printf("Error reading message: %v", err)
// 			return
// 		}
//
// 		switch signalserver.EventType(buf[0]) {
// 		case signalserver.Join:
// 			msg, err := decodeCBOR[signalserver.MessageContent[signalserver.Member]](buf[1:n])
// 			if err != nil {
// 				continue
// 			}
// 			c.handleJoin(msg, onUDP, onTCP)
// 			break
// 		case signalserver.Leave:
// 			msg, err := decodeCBOR[signalserver.MessageContent[any]](buf[1:n])
// 			if err != nil {
// 				continue
// 			}
// 			c.handleLeave(msg)
// 			break
// 		case signalserver.RTCOffer:
// 			msg, err := decodeCBOR[signalserver.MessageContent[signalserver.Offer]](buf[1:n])
// 			if err != nil {
// 				continue
// 			}
// 			c.handleRTCOffer(msg, onUDP, onTCP)
// 			break
// 		case signalserver.RTCAnswer:
// 			msg, err := decodeCBOR[signalserver.MessageContent[signalserver.Offer]](buf[1:n])
// 			if err != nil {
// 				continue
// 			}
// 			c.handleRTCAnswer(msg)
// 			break
// 		case signalserver.RTCICECandidate:
// 			msg, err := decodeCBOR[signalserver.MessageContent[webrtc.ICECandidateInit]](buf[1:n])
// 			if err != nil {
// 				continue
// 			}
// 			c.handleICECandidate(msg)
// 			break
// 		default:
// 			// Do nothing
// 		}
// 	}
// }
//
// func (c *Client) handleJoin(msg signalserver.MessageContent[signalserver.Member], onUDP MessageHandler, onTCP MessageHandler) {
// 	slog.Info("Handling join message",
// 		"id", msg.Content.ID,
// 		"name", msg.Content.Name)
//
// 	_, exist := c.Peers.Get(msg.Content.ID)
// 	if exist {
// 		slog.Warn("Member already exists", "id", msg.Content.ID)
// 		return
// 	}
// 	if msg.Content.ID == c.ID {
// 		return
// 	}
//
// 	peer := c.addPeer(msg.Content, true, onUDP, onTCP)
// 	c.addNewDataChannel(peer, onUDP, onTCP)
// }
//
// func (c *Client) addPeer(member signalserver.Member, isJoinNotRTCOffer bool, onUDP MessageHandler, onTCP MessageHandler) *Peer {
// 	config := webrtc.Configuration{
// 		ICEServers: []webrtc.ICEServer{
// 			// {
// 			// 	URLs: []string{"stun:stun.l.google.com:19302"},
// 			// },
// 			// {
// 			// 	URLs:       []string{"turn:127.0.0.1:3478"},
// 			// 	Username:   "username1",
// 			// 	Credential: "password1",
// 			// },
// 		},
// 	}
//
// 	peerConnection, err := webrtc.NewPeerConnection(config)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	peer := &Peer{
// 		ID:         member.ID,
// 		Name:       member.Name,
// 		Connection: peerConnection,
// 	}
// 	c.Peers.Set(member.ID, peer)
//
// 	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
// 		slog.Debug("ICE Connection State has changed", "peer", member.ID, "state", connectionState.String())
// 	})
//
// 	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
// 		if candidate == nil {
// 			return
// 		}
//
// 		msg := c.newMessage(signalserver.RTCICECandidate, candidate.ToJSON())
// 		msg.To = member.ID
//
// 		if err := c.SendSignal(msg.ToCBOR()); err != nil {
// 			panic(err)
// 		}
// 	})
//
// 	peerConnection.OnNegotiationNeeded(func() {
// 		offer, err := peerConnection.CreateOffer(nil)
// 		if err != nil {
// 			panic(err)
// 		}
//
// 		if err := peerConnection.SetLocalDescription(offer); err != nil {
// 			panic(err)
// 		}
//
// 		if !isJoinNotRTCOffer {
// 			// If this is a message sent first time after joining,
// 			// then we send the offer to invite yourself to join other users.
// 			return
// 		}
//
// 		msg := c.newMessage(signalserver.RTCOffer, signalserver.Offer{
// 			Name:  c.Name,
// 			Offer: offer,
// 		})
// 		msg.To = member.ID
// 		if err := c.SendSignal(msg.ToCBOR()); err != nil {
// 			panic(err)
// 		}
// 	})
//
// 	peerConnection.OnDataChannel(func(channel *webrtc.DataChannel) {
// 		if channel.Label() == "udp" {
// 			channel.OnOpen(func() {
// 				slog.Info("Data channel is open", "channel", channel.Label(), "peer", member.ID)
// 				peer.ChannelUDP = channel
// 			})
// 			channel.OnMessage(func(msg webrtc.DataChannelMessage) {
// 				onUDP(peer, msg)
// 			})
// 		}
// 		if channel.Label() == "tcp" {
// 			channel.OnOpen(func() {
// 				slog.Info("Data channel is open", "channel", channel.Label(), "peer", member.ID)
// 				peer.ChannelTCP = channel
// 			})
// 			channel.OnMessage(func(msg webrtc.DataChannelMessage) {
// 				onTCP(peer, msg)
// 			})
// 		}
// 	})
//
// 	return peer
// }
//
// func (c *Client) addNewDataChannel(peer *Peer, onUDP, onTCP MessageHandler) {
// 	log.Println("Adding new data channel for", peer.ID)
//
// 	{
// 		dc, err := peer.Connection.CreateDataChannel("udp", nil)
// 		if err != nil {
// 			panic(err)
// 		}
// 		peer.ChannelUDP = dc
//
// 		dc.OnError(func(err error) {
// 			slog.Warn("Data channel error", "error", err)
// 		})
//
// 		dc.OnOpen(func() {
// 			peer.ChannelUDP = dc
// 		})
//
// 		dc.OnClose(func() {
// 			log.Printf("dataChannel for %s has closed", peer.ID)
// 			c.Peers.Delete(peer.ID)
// 		})
//
// 		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
// 			onUDP(peer, msg)
// 		})
// 	}
//
// 	{
// 		dc, err := peer.Connection.CreateDataChannel("tcp", nil)
// 		if err != nil {
// 			panic(err)
// 		}
// 		peer.ChannelTCP = dc
//
// 		dc.OnError(func(err error) {
// 			slog.Warn("Data channel error", "error", err)
// 		})
//
// 		dc.OnOpen(func() {
// 			peer.ChannelTCP = dc
// 		})
//
// 		dc.OnClose(func() {
// 			log.Printf("dataChannel for %s has closed", peer.ID)
// 			c.Peers.Delete(peer.ID)
// 		})
//
// 		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
// 			onTCP(peer, msg)
// 		})
// 	}
// }
//
// func (c *Client) handleLeave(msg signalserver.MessageContent[any]) {
// 	peer, ok := c.Peers.Get(msg.From)
// 	if !ok {
// 		slog.Error("Could not find peer")
// 		return
// 	}
//
// 	if peer.ID == c.ID {
// 		return
// 	}
//
// 	log.Printf("User %s left", peer.Name)
// 	c.Peers.Delete(peer.ID)
// }
//
// func (c *Client) handleRTCOffer(msg signalserver.MessageContent[signalserver.Offer], onUDP MessageHandler, onTCP MessageHandler) {
// 	peer := c.addPeer(signalserver.Member{ID: msg.From, Name: msg.Content.Name}, false, onUDP, onTCP)
//
// 	if err := peer.Connection.SetRemoteDescription(msg.Content.Offer); err != nil {
// 		panic(err)
// 	}
//
// 	answer, err := peer.Connection.CreateAnswer(nil)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	if err := peer.Connection.SetLocalDescription(answer); err != nil {
// 		panic(err)
// 	}
//
// 	response := c.newMessage(signalserver.RTCAnswer, signalserver.Offer{Name: c.Name, Offer: answer})
// 	response.To = msg.From
//
// 	if err := c.SendSignal(response.ToCBOR()); err != nil {
// 		panic(err)
// 	}
// }
//
// func (c *Client) handleRTCAnswer(message signalserver.MessageContent[signalserver.Offer]) {
// 	answer := webrtc.SessionDescription{
// 		Type: webrtc.SDPTypeAnswer,
// 		SDP:  message.Content.Offer.SDP,
// 	}
//
// 	peer, ok := c.Peers.Get(message.From)
// 	if !ok {
// 		panic("Could not find peer")
// 	}
//
// 	if err := peer.Connection.SetRemoteDescription(answer); err != nil {
// 		panic(err)
// 	}
// }
//
// func (c *Client) handleICECandidate(message signalserver.MessageContent[webrtc.ICECandidateInit]) {
// 	var candidate = message.Content
//
// 	peer, ok := c.Peers.Get(message.From)
// 	if !ok {
// 		return
// 	}
//
// 	if err := peer.Connection.AddICECandidate(candidate); err != nil {
// 		panic(err)
// 	}
// }
//
// // newMessage creates a new Message instance
// func (c *Client) newMessage(msgType signalserver.EventType, content any) *signalserver.Message {
// 	return &signalserver.Message{
// 		From:    c.ID,
// 		Type:    msgType,
// 		Content: content,
// 	}
// }
//
// func (c *Client) BroadcastUDP(payload []byte) {
// 	c.Peers.Range(func(id string, peer *Peer) {
// 		slog.Debug("Broadcasting message", "to", peer.Name)
//
// 		if peer.ChannelUDP == nil {
// 			slog.Warn("No data channel", "to", peer.Name)
// 			return
// 		}
//
// 		err := peer.ChannelUDP.Send(payload)
// 		if err != nil {
// 			slog.Warn("Error broadcasting message", "to", peer.Name, "error", err)
// 		}
// 		return
// 	})
// }
//
// func (c *Client) BroadcastTCP(payload []byte) {
// 	c.Peers.Range(func(id string, peer *Peer) {
// 		slog.Debug("Broadcasting message", "to", peer.Name)
//
// 		if peer.ChannelTCP == nil {
// 			slog.Warn("No data channel", "to", peer.Name)
// 			return
// 		}
//
// 		err := peer.ChannelTCP.Send(payload)
// 		if err != nil {
// 			slog.Warn("Error broadcasting message", "to", peer.Name, "error", err)
// 		}
// 		return
// 	})
// }
//
// func (c *Client) SendSignal(message []byte) (err error) {
// 	_, err = c.ws.Write(message)
// 	return
// }
