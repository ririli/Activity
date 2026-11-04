package grpc

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	identityv1 "activity/contracts/rpc/identity/v1"
	"activity/pkg/platform"
	"activity/services/identity-service/internal/identity/application"
	"activity/services/identity-service/internal/identity/domain"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeRepo struct {
	names   map[string]string
	err     error
	lastIDs []string
	calls   int
}

func (f *fakeRepo) Register(context.Context, domain.Account, domain.Session, uint64, []byte) error {
	return nil
}
func (f *fakeRepo) AccountByName(context.Context, string) (domain.Account, error) {
	return domain.Account{}, nil
}
func (f *fakeRepo) SessionAccount(context.Context, string) (domain.Session, domain.Account, error) {
	return domain.Session{}, domain.Account{}, nil
}
func (f *fakeRepo) CreateSession(context.Context, domain.Session) error { return nil }
func (f *fakeRepo) Rotate(context.Context, string, [32]byte, [32]byte) (application.Rotation, error) {
	return application.Rotation{}, nil
}
func (f *fakeRepo) Revoke(context.Context, string) (time.Time, error) { return time.Time{}, nil }
func (f *fakeRepo) PublicProfiles(_ context.Context, ids []string) (map[string]string, error) {
	f.calls++
	f.lastIDs = append([]string(nil), ids...)
	return f.names, f.err
}

func TestBatchGetPublicProfilesDeduplicatesAndPreservesOrder(t *testing.T) {
	repo := &fakeRepo{names: map[string]string{"1": "alice", "2": "", "3": "carol"}}
	repo.names["2"] = "bob"
	srv := NewServer(&application.Service{Repo: repo})
	resp, err := srv.BatchGetPublicProfiles(context.Background(), &identityv1.BatchGetPublicProfilesRequest{
		UserIds: []string{"2", "1", "2", "3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Profiles) != 3 {
		t.Fatalf("profiles = %d", len(resp.Profiles))
	}
	if resp.Profiles[0].UserId != "2" || resp.Profiles[1].UserId != "1" || resp.Profiles[2].UserId != "3" {
		t.Fatalf("unexpected order %+v", resp.Profiles)
	}
	if repo.calls != 1 || len(repo.lastIDs) != 3 {
		t.Fatalf("repo calls=%d ids=%v", repo.calls, repo.lastIDs)
	}
}

func TestBatchGetPublicProfilesRejectsInvalidAndTooMany(t *testing.T) {
	srv := NewServer(&application.Service{Repo: &fakeRepo{}})
	if _, err := srv.BatchGetPublicProfiles(context.Background(), &identityv1.BatchGetPublicProfilesRequest{UserIds: []string{"0"}}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid id code = %v", status.Code(err))
	}
	ids := make([]string, 101)
	for i := range ids {
		ids[i] = strconv.Itoa(i + 1)
	}
	if _, err := srv.BatchGetPublicProfiles(context.Background(), &identityv1.BatchGetPublicProfilesRequest{UserIds: ids}); status.Code(err) == codes.OK {
		t.Fatal("expected >100 unique IDs to fail")
	}
}

func TestBatchGetPublicProfilesMapsDependencyError(t *testing.T) {
	srv := NewServer(&application.Service{Repo: &fakeRepo{err: platform.Fail(503, "DEPENDENCY_UNAVAILABLE")}})
	_, err := srv.BatchGetPublicProfiles(context.Background(), &identityv1.BatchGetPublicProfilesRequest{UserIds: []string{"1"}})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("code = %v", status.Code(err))
	}
}

func TestBatchGetPublicProfilesEmpty(t *testing.T) {
	srv := NewServer(&application.Service{Repo: &fakeRepo{err: errors.New("must not be called")}})
	resp, err := srv.BatchGetPublicProfiles(context.Background(), &identityv1.BatchGetPublicProfilesRequest{})
	if err != nil || len(resp.Profiles) != 0 {
		t.Fatalf("empty call = %+v, %v", resp, err)
	}
}

func TestInternalAuth(t *testing.T) {
	t.Setenv("INTERNAL_TOKEN", "01234567890123456789012345678901")
	interceptor := InternalAuth()
	handler := func(context.Context, any) (any, error) { return "ok", nil }
	method := "/activity.identity.v1.IdentityService/BatchGetPublicProfiles"
	info := &grpc.UnaryServerInfo{FullMethod: method}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-internal-token", "01234567890123456789012345678901"))
	if out, err := interceptor(ctx, nil, info, handler); err != nil || out != "ok" {
		t.Fatalf("valid token = %v, %v", out, err)
	}
	if _, err := interceptor(context.Background(), nil, info, handler); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing token code = %v", status.Code(err))
	}
	bad := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-internal-token", "wrong"))
	if _, err := interceptor(bad, nil, info, handler); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("wrong token code = %v", status.Code(err))
	}
	health := &grpc.UnaryServerInfo{FullMethod: "/grpc.health.v1.Health/Check"}
	if out, err := interceptor(context.Background(), nil, health, handler); err != nil || out != "ok" {
		t.Fatalf("health = %v, %v", out, err)
	}
}
