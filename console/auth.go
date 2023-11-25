package console

import "net/http"

// CreateAccount is used by the launcher to create a new account.
func (c *Console) CreateAccount(w http.ResponseWriter, r *http.Request) {
	return
}

// Authenticate is used by the launcher to create a new session.
//
// It should provide information such as the user's id, username, the JWT token, and the session id.
func (c *Console) Authenticate(w http.ResponseWriter, r *http.Request) {
	return
}
