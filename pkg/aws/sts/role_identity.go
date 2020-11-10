package sts

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

type RoleIdentity struct {
	Role        ResolvedRole
	SessionName string
	ExternalID  string
}

func NewRoleIdentity(arnResolver ARNResolver, role, sessionName, externalID string) (*RoleIdentity, error) {
	resolvedRole, err := arnResolver.Resolve(role)
	if err != nil {
		return nil, err
	}

	return &RoleIdentity{
		Role:        *resolvedRole,
		SessionName: sessionName,
		ExternalID:  externalID,
	}, nil
}

func (i *RoleIdentity) String() string {
	return fmt.Sprintf("%s|%s|%s", i.Role.ARN, i.SessionName, i.ExternalID)
}

func (i *RoleIdentity) LogFields() log.Fields {
	return log.Fields{
		"pod.iam.role":    i.Role,
		"pod.iam.roleArn": i.Role.ARN,
	}
}
