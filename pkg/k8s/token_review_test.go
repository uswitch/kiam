package k8s

import (
	"testing"

	auth "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MockReviewer struct {
	*TokenReviewer
	data map[string]*KiamReview
}

func (m *MockReviewer) convert() *TokenReviewer {
	return m.TokenReviewer
}

func MockData() map[string]*KiamReview {
	return map[string]*KiamReview{
		"foo": &KiamReview{
			TokenReview: &auth.TokenReview{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
				Status: auth.TokenReviewStatus{
					Authenticated: true,
					User: auth.UserInfo{
						Username: "system:serviceaccount:bar:foo",
						UID:      "thecakeisalie",
					},
				},
			},
		},
	}
}

func (m *MockReviewer) CreateReview(token string) (*KiamReview, error) {
	review := m.data[token]
	if review != nil {
		review.TokenReviewer = &TokenReviewer{m.Client, m.Namespace, m.Name}
		return review, nil
	}
	noAuthReview := &auth.TokenReview{
		Status: auth.TokenReviewStatus{
			Authenticated: false,
		},
	}
	review = &KiamReview{TokenReview: noAuthReview, TokenReviewer: m.convert()}

	return review, nil
}

func NewMockReviewer(namespace, name string) *MockReviewer {
	return &MockReviewer{
		&TokenReviewer{
			Namespace: namespace,
			Name:      name,
		},
		MockData(),
	}
}

func TestReviewEmptyToken(t *testing.T) {
	reviewer := NewMockReviewer("", "")
	review, err := reviewer.CreateReview("")
	if err != nil {
		t.Fatal("empty token should not be an error")
	}
	auth, _ := review.Review()
	if auth == true {
		t.Fatal("empty token should not be valid")
	}
}

func TestReviewNoNamespace(t *testing.T) {
	reviewer := NewMockReviewer("", "")
	review, _ := reviewer.CreateReview("foo")
	res, err := review.Review()
	if err != nil {
		t.Fatalf("valid token should not return error %v", err)
	}
	if res != true {
		t.Fatalf("should have been valid: %v error: %v", res, err)
	}
}

func TestReviewWithNameContexts(t *testing.T) {
	reviewer := NewMockReviewer("bar", "foo")
	review, _ := reviewer.CreateReview("foo")
	res, err := review.Review()
	if err != nil {
		t.Fatalf("valid token should not return error %v", err)
	}
	if res != true {
		t.Fatalf("should have been valid: %v error: %v", res, err)
	}
}
