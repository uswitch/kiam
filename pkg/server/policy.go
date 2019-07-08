// Copyright 2017 uSwitch
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package server

import (
	"context"
	"fmt"
	"regexp"

	glob "github.com/ryanuber/go-glob"
	"github.com/uswitch/kiam/pkg/aws/sts"
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
	pods     k8s.PodGetter
	resolver sts.ARNResolver
}

func NewRequestingAnnotatedRolePolicy(p k8s.PodGetter, resolver sts.ARNResolver) *RequestingAnnotatedRolePolicy {
	return &RequestingAnnotatedRolePolicy{pods: p, resolver: resolver}
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
	pod, err := p.pods.GetPodByIP(podIP)
	if err != nil {
		return nil, err
	}

	annotatedRole := p.resolver.Resolve(k8s.PodRole(pod))
	role = p.resolver.Resolve(role)

	if annotatedRole != role {
		return &forbidden{requested: role, annotated: annotatedRole}, nil
	}

	return &allowed{}, nil
}

type NamespacePermittedRoleNamePolicy struct {
	namespaces k8s.NamespaceFinder
	pods       k8s.PodGetter
}

func NewNamespacePermittedRoleNamePolicy(n k8s.NamespaceFinder, p k8s.PodGetter) *NamespacePermittedRoleNamePolicy {
	return &NamespacePermittedRoleNamePolicy{namespaces: n, pods: p}
}

type namespacePolicyForbidden struct {
	expression string
	role       string
}

func (f *namespacePolicyForbidden) IsAllowed() bool {
	return false
}

func (f *namespacePolicyForbidden) Explanation() string {
	return fmt.Sprintf("namespace policy expression '%s' forbids role '%s'", f.expression, f.role)
}

func (p *NamespacePermittedRoleNamePolicy) IsAllowedAssumeRole(ctx context.Context, role, podIP string) (Decision, error) {

	pod, err := p.pods.GetPodByIP(podIP)
	if err != nil {
		return nil, err
	}

	ns, err := p.namespaces.FindNamespace(ctx, pod.GetObjectMeta().GetNamespace())
	if err != nil {
		return nil, err
	}

	// Regex version, if we have one use it
	expression := ns.GetAnnotations()[k8s.AnnotationPermittedKey]
	if expression != "" {
		re, err := regexp.Compile(expression)
		if err != nil {
			return nil, err
		}

		if re.MatchString(role) {
			return &allowed{}, nil
		}
	}

	// If we have other listed patterns, let use them
	for _, rolePattern := range k8s.GetNamespaceRoleAnnotation(ns, k8s.AnnotationAllowedRoles) {
		if glob.Glob(rolePattern, role) {
			return &allowed{}, nil
		}
	}

	return &namespacePolicyForbidden{
		expression: fmt.Sprintf("(%s=%s %s=%s)",
			k8s.AnnotationPermittedKey, ns.GetAnnotations()[k8s.AnnotationPermittedKey],
			k8s.AnnotationAllowedRoles, ns.GetAnnotations()[k8s.AnnotationAllowedRoles],
		),
		role: role,
	}, nil
}
