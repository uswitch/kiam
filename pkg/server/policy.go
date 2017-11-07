package server

import (
	"context"
	"fmt"
	"github.com/uswitch/kiam/pkg/k8s"
	pb "github.com/uswitch/kiam/proto"
)

type adaptedDecision struct {
	d *pb.Decision
}

func (a *adaptedDecision) IsAllowed() bool {
	return a.d.IsAllowed
}

func (a *adaptedDecision) Explanation() string {
	return a.d.Explanation
}

// Decision reports (with message) as to whether the assume role is permitted.
type Decision interface {
	IsAllowed() bool
	Explanation() string
}

type allowed struct {
}

func (a *allowed) IsAllowed() bool {
	return true
}
func (a *allowed) Explanation() string {
	return ""
}

// AssumeRolePolicy allows for policy to check whether pods can assume the role being
// requested
type AssumeRolePolicy interface {
	IsAllowedAssumeRole(ctx context.Context, roleName, podIP string) (Decision, error)
}

// CompositeAssumeRolePolicy allows multiple policies to be checked
type CompositeAssumeRolePolicy struct {
	policies []AssumeRolePolicy
}

func (p *CompositeAssumeRolePolicy) IsAllowedAssumeRole(ctx context.Context, role, podIP string) (Decision, error) {
	for _, policy := range p.policies {
		decision, err := policy.IsAllowedAssumeRole(ctx, role, podIP)
		if err != nil {
			return nil, err
		}
		if !decision.IsAllowed() {
			return decision, nil
		}
	}

	return &allowed{}, nil
}

// Creates a AssumeRolePolicy that tests all policies pass.
func Policies(p ...AssumeRolePolicy) *CompositeAssumeRolePolicy {
	return &CompositeAssumeRolePolicy{
		policies: p,
	}
}

// RequestingAnnotatedRolePolicy ensures the pod is requesting the role that it's
// currently annotated with.
type RequestingAnnotatedRolePolicy struct {
	pods k8s.RoleFinder
}

func NewRequestingAnnotatedRolePolicy(finder k8s.RoleFinder) *RequestingAnnotatedRolePolicy {
	return &RequestingAnnotatedRolePolicy{pods: finder}
}

type forbidden struct {
	requested string
	annotated string
}

func (f *forbidden) IsAllowed() bool {
	return false
}
func (f *forbidden) Explanation() string {
	return fmt.Sprintf("requested '%s' but annotated with '%s', forbidden", f.requested, f.annotated)
}

func (p *RequestingAnnotatedRolePolicy) IsAllowedAssumeRole(ctx context.Context, role, podIP string) (Decision, error) {
	annotatedRole, err := p.pods.FindRoleFromIP(ctx, podIP)
	if err != nil {
		return nil, err
	}

	if annotatedRole != role {
		return &forbidden{requested: role, annotated: annotatedRole}, nil
	}

	return &allowed{}, nil
}

type NamespacePermittedRoleNamePolicy struct {
	namespaces k8s.NamespaceFinder
	pods       k8s.RoleFinder
}

func NewNamespacePermittedRoleNamePolicy(n k8s.NamespaceFinder, p k8s.RoleFinder) *NamespacePermittedRoleNamePolicy {
	return &NamespacePermittedRoleNamePolicy{namespaces: n, pods: p}
}

func (p *NamespacePermittedRoleNamePolicy) IsAllowedAssumeRole(ctx context.Context, role, podIP string) (Decision, error) {
	return nil, nil
}
