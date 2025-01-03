package backend

import (
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet/command"
	"github.com/dimspell/gladiator/internal/console/database"
)

// HandleListChannels handles 0xbff (255-11) command
func (b *Backend) HandleListChannels(session *bsession.Session, req ListChannelsRequest) error {
	var response []byte
	for _, channel := range database.Channels {
		response = append(response, channel...)
		response = append(response, 0)
	}
	return session.Send(command.ListChannels, response)
}

type ListChannelsRequest []byte
