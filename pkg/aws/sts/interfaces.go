package sts

import (
	"context"
)

type CredentialsProvider interface {
	CredentialsForRole(ctx context.Context, role string) (*Credentials, error)
}

type CredentialsCache interface {
	CredentialsForRole(ctx context.Context, role string) (*Credentials, error)
	Expiring() chan *RoleCredentials
}
