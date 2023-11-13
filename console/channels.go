package console

import (
	"encoding/json"
	"net/http"

	"github.com/dispel-re/dispel-multi/internal/database/sqlite"
)

func (c *Console) ListChannels(w http.ResponseWriter, r *http.Request) {
	// Get channels from database
	channels := sqlite.Channels

	// Write channels to response
	enc := json.NewEncoder(w)
	if err := enc.Encode(channels); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
