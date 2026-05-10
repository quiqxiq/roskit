// Package testhelpers provides shared mock implementations for API handler tests.
package testhelpers

import (
	"context"
	"errors"

	routeros "github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

// MockUserRepository implements services.UserRepository for auth handler tests.
type MockUserRepository struct {
	User      *models.User
	Err       error
	UserCount int64
}

func (m *MockUserRepository) Create(_ context.Context, user *models.User) error {
	if m.Err != nil {
		return m.Err
	}
	m.User = user
	return nil
}

func (m *MockUserRepository) GetByID(_ context.Context, _ uint) (*models.User, error) {
	return m.User, m.Err
}

func (m *MockUserRepository) GetByUsername(_ context.Context, _ string) (*models.User, error) {
	return m.User, m.Err
}

func (m *MockUserRepository) List(_ context.Context) ([]*models.User, error) {
	if m.User == nil {
		return nil, m.Err
	}
	return []*models.User{m.User}, m.Err
}

func (m *MockUserRepository) Update(_ context.Context, _ *models.User) error {
	return m.Err
}

func (m *MockUserRepository) UpdateLastLogin(_ context.Context, _ uint) error {
	return m.Err
}

func (m *MockUserRepository) Delete(_ context.Context, _ uint) error {
	return m.Err
}

func (m *MockUserRepository) Count(_ context.Context) (int64, error) {
	return m.UserCount, m.Err
}

// MockDispatcher implements behavior.Dispatcher for Bridge-backed service tests.
type MockDispatcher struct {
	QueryRows   []map[string]string
	QueryErr    error
	MutateReply *routeros.Reply
	MutateErr   error
}

func (d *MockDispatcher) Meta(path string) *command.CommandMeta {
	return &command.CommandMeta{
		Path:        path,
		Measurement: "mock",
		Type:        command.CommandTypeQuery,
	}
}

func (d *MockDispatcher) Stream(_ context.Context, _, _ string) (context.CancelFunc, error) {
	return func() {}, nil
}

func (d *MockDispatcher) StopStream(_, _ string) {}

func (d *MockDispatcher) Query(_ context.Context, _, _ string, _ ...string) ([]map[string]string, error) {
	return d.QueryRows, d.QueryErr
}

func (d *MockDispatcher) Mutate(_ context.Context, _, _ string, _ ...string) (*routeros.Reply, error) {
	if d.MutateErr != nil {
		return nil, d.MutateErr
	}
	if d.MutateReply != nil {
		return d.MutateReply, nil
	}
	return &routeros.Reply{}, nil
}

var _ behavior.Dispatcher = (*MockDispatcher)(nil)

// ErrNotFound is a sentinel used by mocks to simulate not-found scenarios.
var ErrNotFound = errors.New("record not found")
