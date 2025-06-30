package console

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

// Adapted from: github.com/go-chi/render

// ctxKeyStatus is a context key to record a future HTTP response status code.
var ctxKeyStatus = &struct{}{}

// withStatus sets a HTTP response status code hint into request context at any point
// during the request life-cycle. Before the Responder sends its response header
// it will check the StatusCtxKey
func withStatus(r *http.Request, status int) {
	*r = *r.WithContext(context.WithValue(r.Context(), ctxKeyStatus, status))
}

// renderJSON marshals 'v' to JSON, automatically escaping HTML and setting the
// Content-Type as application/json.
func renderJSON(w http.ResponseWriter, r *http.Request, document any) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(document); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if status, ok := r.Context().Value(ctxKeyStatus).(int); ok {
		w.WriteHeader(status)
	}
	w.Write(buf.Bytes()) //nolint:errcheck
}
