package grpc

import (
	"context"
	"database/sql"
	"errors"

	identityv1 "activity/contracts/rpc/identity/v1"
	"activity/pkg/platform"
	"activity/services/identity-service/internal/identity/application"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server adapts the identity application service to the internal profile RPC.
type Server struct {
	identityv1.UnimplementedIdentityServiceServer
	App *application.Service
}

func NewServer(app *application.Service) *Server { return &Server{App: app} }

func Register(s grpc.ServiceRegistrar, app *application.Service) {
	identityv1.RegisterIdentityServiceServer(s, NewServer(app))
}

func (s *Server) BatchGetPublicProfiles(ctx context.Context, req *identityv1.BatchGetPublicProfilesRequest) (*identityv1.BatchGetPublicProfilesResponse, error) {
	if s.App == nil {
		return nil, status.Error(codes.Unavailable, "identity not ready")
	}
	profiles, err := s.App.PublicProfiles(ctx, req.GetUserIds())
	if err != nil {
		return nil, grpcError(err)
	}
	out := &identityv1.BatchGetPublicProfilesResponse{Profiles: make([]*identityv1.PublicProfile, 0, len(profiles))}
	for _, p := range profiles {
		out.Profiles = append(out.Profiles, &identityv1.PublicProfile{UserId: p.UserID, DisplayName: p.DisplayName})
	}
	return out, nil
}

func grpcError(err error) error {
	var fault *platform.Fault
	if errors.As(err, &fault) {
		code := codes.Internal
		switch fault.Status {
		case 400:
			code = codes.InvalidArgument
		case 401:
			code = codes.Unauthenticated
		case 403:
			code = codes.PermissionDenied
		case 404:
			code = codes.NotFound
		case 503:
			code = codes.Unavailable
		}
		return status.Error(code, fault.Code)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return status.Error(codes.NotFound, "not found")
	}
	return status.Error(codes.Internal, "internal")
}
