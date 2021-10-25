package sts

import (
	"fmt"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
)

type RoleIdentity struct {
	Role        ResolvedRole
	SessionName string
	ExternalID  string
	SessionTags map[string]string
}

func NewRoleIdentity(arnResolver ARNResolver, role, sessionName, externalID string, sessionTags map[string]string) (*RoleIdentity, error) {
	resolvedRole, err := arnResolver.Resolve(role)
	if err != nil {
		return nil, err
	}

	return &RoleIdentity{
		Role:        *resolvedRole,
		SessionName: sessionName,
		ExternalID:  externalID,
		SessionTags: sessionTags,
	}, nil
}

func (i *RoleIdentity) String() string {
	s := fmt.Sprintf("%s|%s|%s", i.Role.ARN, i.SessionName, i.ExternalID)
	if len(i.SessionTags) > 0 {
		var kvp []string
		for k, v := range i.SessionTags {
			kvp = append(kvp, fmt.Sprintf("%s:%s", k, v))
		}
		// Looping through maps has non-deterministic ordering.
		sort.Strings(kvp)
		s += fmt.Sprintf("|%s", strings.Join(kvp, ","))
	}
	return s
}

func (i *RoleIdentity) LogFields() log.Fields {
	return log.Fields{
		"pod.iam.role":    i.Role,
		"pod.iam.roleArn": i.Role.ARN,
	}
}
