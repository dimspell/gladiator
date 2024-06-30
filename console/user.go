package console

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/dimspell/gladiator/console/auth"
	"github.com/dimspell/gladiator/console/database"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
)

var _ multiv1connect.UserServiceHandler = (*userServiceServer)(nil)

type userServiceServer struct {
	DB      *database.SQLite
	Queries *database.Queries
}

// CreateUser creates a new user.
func (s *userServiceServer) CreateUser(ctx context.Context, req *connect.Request[multiv1.CreateUserRequest]) (*connect.Response[multiv1.CreateUserResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	password, err := auth.NewPassword(req.Msg.Password)
	if err != nil {
		slog.Warn("could not hash the password", "err", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	user, err := s.Queries.CreateUser(ctx, database.CreateUserParams{
		Username: req.Msg.Username,
		Password: password.String(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
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

	user, err := s.Queries.GetUserByName(ctx, req.Msg.Username)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
		// slog.Debug("packet-41: could not find a user", "username", data.Username)
		// return b.Send(session.Conn, ClientAuthentication, []byte{0, 0, 0, 0})
	}

	if !auth.CheckPassword(req.Msg.Password, user.Password) {
		// slog.Debug("packet-41: incorrect password")
		// return b.Send(session.Conn, ClientAuthentication, []byte{0, 0, 0, 0})
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("incorrect password"))
	}

	resp := connect.NewResponse(&multiv1.AuthenticateUserResponse{
		User: &multiv1.User{
			UserId:   user.ID,
			Username: user.Username,
		}},
	)
	return resp, nil
}

// GetUser gets a user by ID.
func (s *userServiceServer) GetUser(ctx context.Context, req *connect.Request[multiv1.GetUserRequest]) (*connect.Response[multiv1.GetUserResponse], error) {
	user, err := s.Queries.GetUserByID(ctx, req.Msg.UserId)
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
