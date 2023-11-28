package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dispel-re/dispel-multi/console/dto"
	"github.com/go-chi/chi/v5"
)

// ListCharacters lists all characters for a given user.
func (c *ConsoleServer) ListCharacters(w http.ResponseWriter, r *http.Request) {
	userId, err := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := c.DB.GetUserByID(r.Context(), userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	characters, err := c.DB.ListCharacters(r.Context(), userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]dto.Character, 0, len(characters))
	for _, character := range characters {
		response = append(response, dto.Character{
			UserId:        user.ID,
			UserName:      user.Username,
			CharacterId:   character.ID,
			CharacterName: character.CharacterName,
		})
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(characters); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
