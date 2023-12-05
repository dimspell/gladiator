package console

import (
	"context"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/gen/multi/v1/multiv1connect"
	"github.com/dispel-re/dispel-multi/internal/database"
)

type characterServiceServer struct {
	multiv1connect.UnimplementedCharacterServiceHandler

	DB *database.Queries
}

func (s *characterServiceServer) ListCharacters(
	ctx context.Context,
	req *connect.Request[multiv1.ListCharactersRequest],
) (*connect.Response[multiv1.ListCharactersResponse], error) {
	// user, err := c.DB.GetUserByID(r.Context(), userId)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusNotFound)
	// 	return
	// }
	//
	// characters, err := c.DB.ListCharacters(r.Context(), userId)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	//
	// response := make([]dto.Character, 0, len(characters))
	// for _, character := range characters {
	// 	response = append(response, dto.Character{
	// 		UserId:        user.ID,
	// 		UserName:      user.Username,
	// 		CharacterId:   character.ID,
	// 		CharacterName: character.CharacterName,
	// 	})
	// }

	// characters, err := b.DB.ListCharacters(context.TODO(), session.UserID)
	return nil, nil
}

func (s *characterServiceServer) GetCharacter(
	ctx context.Context,
	req *connect.Request[multiv1.GetCharacterRequest],
) (*connect.Response[multiv1.GetCharacterResponse], error) {
	// character, err := b.DB.FindCharacter(context.TODO(), database.FindCharacterParams{
	// 	UserID:        session.UserID,
	// 	CharacterName: data.CharacterName,
	// })
	return nil, nil
}

func (s *characterServiceServer) DeleteCharacter(
	ctx context.Context,
	req *connect.Request[multiv1.DeleteCharacterRequest],
) (*connect.Response[multiv1.DeleteCharacterResponse], error) {
	// characters, err := b.DB.DeleteCharacter(context.TODO(), session.UserID)
	return nil, nil
}
