package console

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/dispel-re/dispel-multi/console/database"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/gen/multi/v1/multiv1connect"
	"golang.org/x/crypto/bcrypt"
)

var _ multiv1connect.UserServiceHandler = (*userServiceServer)(nil)

type userServiceServer struct {
	DB *database.Queries
}

func (s *userServiceServer) CreateUser(ctx context.Context, req *connect.Request[multiv1.CreateUserRequest]) (*connect.Response[multiv1.CreateUserResponse], error) {
	password, err := hashPassword(req.Msg.Password)
	if err != nil {
		slog.Warn("packet-42: could not hash the password", "err", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	user, err := s.DB.CreateUser(ctx, database.CreateUserParams{
		Username: req.Msg.Username,
		Password: password,
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

func (s *userServiceServer) AuthenticateUser(ctx context.Context, req *connect.Request[multiv1.AuthenticateUserRequest]) (*connect.Response[multiv1.AuthenticateUserResponse], error) {
	user, err := s.DB.GetUserByName(ctx, req.Msg.Username)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
		// slog.Debug("packet-41: could not find a user", "username", data.Username)
		// return b.Send(session.Conn, ClientAuthentication, []byte{0, 0, 0, 0})
	}

	if !checkPasswordHash(req.Msg.Password, user.Password) {
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

func (s *userServiceServer) GetUser(ctx context.Context, req *connect.Request[multiv1.GetUserRequest]) (*connect.Response[multiv1.GetUserResponse], error) {
	user, err := s.DB.GetUserByID(ctx, req.Msg.UserId)
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

// TODO: Use salt and pepper
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
