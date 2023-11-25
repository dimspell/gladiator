package console

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// ListCharacters lists all characters for a given user.
func (c *Console) ListCharacters(w http.ResponseWriter, r *http.Request) {
	userId, err := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	characters, err := c.DB.ListCharacters(r.Context(), userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(characters); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
