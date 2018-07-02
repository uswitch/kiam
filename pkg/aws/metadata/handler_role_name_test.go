package metadata

import (
	"context"
	"fmt"
	st "github.com/uswitch/kiam/pkg/testutil/server"
	"net/http"
	"net/http/httptest"
	"testing"
)

func clientIP(r *http.Request) (string, error) {
	return "", nil
}

func TestReturnRoleWhenClientResponds(t *testing.T) {
	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/", nil)
	rr := httptest.NewRecorder()
	handler := &roleHandler{client: &st.StubClient{Roles: []st.GetRoleResult{st.GetRoleResult{"foo_role", nil}}}, clientIP: clientIP}

	handler.Handle(context.Background(), rr, r)

	status := rr.Code
	if status != http.StatusOK {
		t.Error("expected 200 response, was", status)
	}
	body := rr.Body.String()
	if body != "foo_role" {
		t.Error("expected foo_role in body, was", body)
	}
}

func TestReturnRoleWhenRetryingFollowingError(t *testing.T) {
	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/", nil)
	rr := httptest.NewRecorder()
	handler := &roleHandler{client: &st.StubClient{Roles: []st.GetRoleResult{st.GetRoleResult{"", fmt.Errorf("unexpected error")}, st.GetRoleResult{"foo_role", nil}}}, clientIP: clientIP}

	handler.Handle(context.Background(), rr, r)

	status := rr.Code
	if status != http.StatusOK {
		t.Error("expected 200 response, was", status)
	}
	body := rr.Body.String()
	if body != "foo_role" {
		t.Error("expected foo_role in body, was", body)
	}
}

func TestReturnsEmptyRoleWhenClientSucceedsWithEmptyRole(t *testing.T) {
	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/", nil)
	rr := httptest.NewRecorder()
	handler := &roleHandler{client: &st.StubClient{Roles: []st.GetRoleResult{st.GetRoleResult{"", nil}}}, clientIP: clientIP}

	handler.Handle(context.Background(), rr, r)

	status := rr.Code
	if status != http.StatusOK {
		t.Error("expected 200 response, was", status)
	}
	body := rr.Body.String()
	if body != "" {
		t.Error("expected empty role in body, was", body)
	}
}
