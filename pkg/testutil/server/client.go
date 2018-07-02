package server

import (
	"context"
	"fmt"
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
	fmt.Printf("idx: %d", c.rolesCallCount)
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
