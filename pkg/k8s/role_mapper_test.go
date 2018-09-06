package k8s

import (
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRoleMapper(t *testing.T) {
	roleMapper := NewRoleMapper(map[string]string{
		"external-dns": "stack-12345-external-dns-98765",
	})

	cases := []struct {
		description string
		pod         *v1.Pod
		expected    string
	}{
		{
			description: "No role",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			expected: "",
		},
		{
			description: "Static role",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"iam.amazonaws.com/role": "static-role",
					},
				},
			},
			expected: "static-role",
		},
		{
			description: "Aliased role",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"iam.amazonaws.com/role-alias": "external-dns",
					},
				},
			},
			expected: "stack-12345-external-dns-98765",
		},
		{
			description: "Both static and alias (static takes precedence)",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"iam.amazonaws.com/role":       "static-role",
						"iam.amazonaws.com/role-alias": "external-dns",
					},
				},
			},
			expected: "static-role",
		},
		{
			description: "Invalid alias",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"iam.amazonaws.com/role-alias": "invalid",
					},
				},
			},
			expected: "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			role := roleMapper.PodRole(tc.pod)
			if role != tc.expected {
				t.Fatalf("got %v, want %v", role, tc.expected)
			}
		})
	}

}
