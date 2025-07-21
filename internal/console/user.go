package console

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/console/auth"
	"github.com/dimspell/gladiator/internal/console/database"
)

var _ multiv1connect.UserServiceHandler = (*userServiceServer)(nil)

type userServiceServer struct {
	DB *database.SQLite
}

// CreateUser creates a new user.
func (s *userServiceServer) CreateUser(ctx context.Context, req *connect.Request[multiv1.CreateUserRequest]) (*connect.Response[multiv1.CreateUserResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	password, err := auth.NewPassword(req.Msg.Password)
	if err != nil {
		slog.Warn("could not hash the password", logging.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	tx, queries, err := s.DB.WithTx(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	user, err := queries.CreateUser(ctx, database.CreateUserParams{
		Username: req.Msg.Username,
		Password: password.String(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeAborted, errors.Join(err, tx.Rollback()))
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}

	resp := connect.NewResponse(&multiv1.CreateUserResponse{
		User: &multiv1.User{
			UserId:   user.ID,
			Username: user.Username,
		}},
	)
	return resp, nil
}

// AuthenticateUser authenticates a user.
func (s *userServiceServer) AuthenticateUser(ctx context.Context, req *connect.Request[multiv1.AuthenticateUserRequest]) (*connect.Response[multiv1.AuthenticateUserResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	user, err := s.DB.Read.GetUserByName(ctx, req.Msg.Username)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("incorrect password or username"))
		// slog.Debug("packet-41: could not find a user", "username", data.Username)
		// return session.SendToGame(ClientAuthentication, []byte{0, 0, 0, 0})
	}

	if !auth.CheckPassword(req.Msg.Password, user.Password) {
		// slog.Debug("packet-41: incorrect password")
		// return session.SendToGame(ClientAuthentication, []byte{0, 0, 0, 0})
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("incorrect password or username"))
	}

	// TODO: pass the secret to generate the token
	// token, err := generateJWT(user.ID)
	// if err != nil {
	// 	return nil, connect.NewError(connect.CodeInternal, err)
	// }

	resp := connect.NewResponse(&multiv1.AuthenticateUserResponse{
		User: &multiv1.User{
			UserId:   user.ID,
			Username: user.Username,
		},
		// Token: token,
	},
	)
	return resp, nil
}

// GetUser gets a user by ID.
func (s *userServiceServer) GetUser(ctx context.Context, req *connect.Request[multiv1.GetUserRequest]) (*connect.Response[multiv1.GetUserResponse], error) {
	user, err := s.DB.Read.GetUserByID(ctx, req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	resp := connect.NewResponse(&multiv1.GetUserResponse{
		User: &multiv1.User{
			UserId:   user.ID,
			Username: user.Username,
		}},
	)
	return resp, nil
}
