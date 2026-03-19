package authsvc

import (
	"context"

	authpb "github.com/dragodui/my-deploy/internal/shared/proto/authpb/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthGRPCServer struct {
	authpb.UnimplementedAuthInternalServer
	Repo *UserRepository
}

func (s *AuthGRPCServer) ValidateUser(ctx context.Context, req *authpb.ValidateUserRequest) (*authpb.UserResponse, error) {
	user, err := s.Repo.GetByID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return &authpb.UserResponse{
		Id:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}, nil
}
