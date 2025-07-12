package console

import (
	"testing"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/stretchr/testify/assert"
)

func TestUserServiceHandler(t *testing.T) {
	t.Run("create user and sign in", func(t *testing.T) {
		service := &userServiceServer{DB: setupDatabase(t)}

		res, err := service.CreateUser(t.Context(), connect.NewRequest(&multiv1.CreateUserRequest{
			Username: "testuser",
			Password: "password",
		}))
		if err != nil {
			t.Fatalf("create user failed: %v", err)
			return
		}

		service.CreateUser(t.Context(), connect.NewRequest(&multiv1.CreateUserRequest{
			Username: "ardmin",
			Password: "password",
		}))

		res2, err := service.AuthenticateUser(t.Context(), connect.NewRequest(&multiv1.AuthenticateUserRequest{
			Username: "testuser",
			Password: "password",
		}))
		if err != nil {
			t.Fatalf("authenticate user failed: %v", err)
			return
		}

		res3, err := service.GetUser(t.Context(), connect.NewRequest(&multiv1.GetUserRequest{
			UserId: 1,
		}))

		assert.Equal(t, int64(1), res.Msg.User.UserId)
		assert.Equal(t, "testuser", res.Msg.User.Username)
		assert.Equal(t, int64(1), res2.Msg.User.UserId)
		assert.Equal(t, "testuser", res2.Msg.User.Username)
		assert.Equal(t, int64(1), res3.Msg.User.UserId)
		assert.Equal(t, "testuser", res3.Msg.User.Username)
	})
}
