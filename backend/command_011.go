package backend

import (
	"github.com/dispel-re/dispel-multi/internal/database/sqlite"
	"github.com/dispel-re/dispel-multi/model"
)

// HandleListChannels handles 0xbff (255-11) command
func (b *Backend) HandleListChannels(session *model.Session, req ListChannelsRequest) error {
	var response []byte
	for _, channel := range sqlite.Channels {
		response = append(response, channel...)
		response = append(response, 0)
	}
	return b.Send(session.Conn, ListChannels, response)
}

type ListChannelsRequest []byte
