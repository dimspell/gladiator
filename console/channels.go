package console

import (
	"encoding/json"
	"net/http"
)

func (c *Console) ListChannels(w http.ResponseWriter, r *http.Request) {
	// Get channels from database
	channels, err := c.DB.ListChannels()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write channels to response
	enc := json.NewEncoder(w)
	if err := enc.Encode(channels); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
