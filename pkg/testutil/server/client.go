package server

import (
	"context"
	"github.com/uswitch/kiam/pkg/aws/sts"
)

// StubClient returns fake server responses
type StubClient struct {
	Roles          []GetRoleResult
	rolesCallCount int
}

// GetRoleResult is a return value from GetRole
type GetRoleResult struct {
	Role  string
	Error error
}

func (c *StubClient) GetRole(ctx context.Context, ip string) (string, error) {
	if c.rolesCallCount == len(c.Roles) {
		v := c.Roles[len(c.Roles)-1]
		return v.Role, v.Error
	}

	currentVal := c.Roles[c.rolesCallCount]
	c.rolesCallCount = c.rolesCallCount + 1

	return currentVal.Role, currentVal.Error
}
func (c *StubClient) GetCredentials(ctx context.Context, ip, role string) (*sts.Credentials, error) {
	return nil, nil
}
func (c *StubClient) Health(ctx context.Context) (string, error) {
	return "ok", nil
}

func (c *StubClient) WithRoles(roles ...GetRoleResult) *StubClient {
	c.Roles = roles
	return c
}

func NewStubClient() *StubClient {
	return &StubClient{}
}
