package k8s

import "k8s.io/api/core/v1"

// AnnotationIAMRoleKey is the key for the annotation specifying the IAM Role
const AnnotationIAMRoleKey = "iam.amazonaws.com/role"

// AnnotationIAMRoleAliasKey is the key for the annotation specifying the IAM
// Role Alias. RoleMapper will map the role alias to the full role name.
const AnnotationIAMRoleAliasKey = "iam.amazonaws.com/role-alias"

// RoleMapper maps a pod to the role it is configured to assume. It supports
// role aliases.
type RoleMapper struct {
	// A map of role aliases to the full role name
	aliases map[string]string
}

// NewRoleMapper creates a new role mapper with the given set of role aliases
func NewRoleMapper(aliases map[string]string) *RoleMapper {
	return &RoleMapper{
		aliases: aliases,
	}
}

// PodRole returns the IAM role specified in the annotation for the Pod
func (m *RoleMapper) PodRole(pod *v1.Pod) string {
	role, ok := pod.ObjectMeta.Annotations[AnnotationIAMRoleKey]
	if ok {
		return role
	}

	alias, ok := pod.ObjectMeta.Annotations[AnnotationIAMRoleAliasKey]
	if ok {
		return m.aliases[alias]
	}

	return ""
}
