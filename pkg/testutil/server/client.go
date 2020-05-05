package server

import (
	"context"
	"github.com/uswitch/kiam/pkg/aws/sts"
)

// StubClient returns fake server responses
type StubClient struct {
	credentials          []GetCredentialsResult
	credentialsCallCount int
	roles                []GetRoleResult
	rolesCallCount       int
	health               string
}

// GetRoleResult is a return value from GetRole
type GetRoleResult struct {
	Role  string
	Error error
}

func (c *StubClient) GetRole(ctx context.Context, ip string) (string, error) {
	if c.rolesCallCount == len(c.roles) {
		v := c.roles[len(c.roles)-1]
		return v.Role, v.Error
	}

	currentVal := c.roles[c.rolesCallCount]
	c.rolesCallCount = c.rolesCallCount + 1

	return currentVal.Role, currentVal.Error
}
func (c *StubClient) GetCredentials(ctx context.Context, ip, role string) (*sts.Credentials, error) {
	if c.credentialsCallCount == len(c.credentials) {
		v := c.credentials[len(c.credentials)-1]
		return v.Credentials, v.Error
	}
	v := c.credentials[c.credentialsCallCount]
	c.credentialsCallCount = c.credentialsCallCount + 1

	return v.Credentials, v.Error
}

func (c *StubClient) Health(ctx context.Context) (string, error) {
	return c.health, nil
}

func (c *StubClient) WithRoles(roles ...GetRoleResult) *StubClient {
	c.roles = roles
	return c
}

func (c *StubClient) WithHealth(health string) *StubClient {
	c.health = health
	return c
}

type GetCredentialsResult struct {
	Credentials *sts.Credentials
	Error       error
}

func (c *StubClient) WithCredentials(credentials ...GetCredentialsResult) *StubClient {
	c.credentials = credentials
	return c
}

func NewStubClient() *StubClient {
	return &StubClient{}
}
