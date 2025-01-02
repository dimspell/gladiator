package backend

import (
	"github.com/dimspell/gladiator/internal/console/database"
)

// HandleListChannels handles 0xbff (255-11) command
func (b *Backend) HandleListChannels(session *Session, req ListChannelsRequest) error {
	var response []byte
	for _, channel := range database.Channels {
		response = append(response, channel...)
		response = append(response, 0)
	}
	return session.Send(ListChannels, response)
}

type ListChannelsRequest []byte
