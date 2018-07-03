package metadata

import (
	"context"
	"fmt"
	"github.com/uswitch/kiam/pkg/server"
	st "github.com/uswitch/kiam/pkg/testutil/server"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReturnRoleWhenClientResponds(t *testing.T) {
	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/", nil)
	rr := httptest.NewRecorder()
	handler := newHandler(st.NewStubClient().WithRoles(st.GetRoleResult{"foo_role", nil}))

	status, err := handler.Handle(context.Background(), rr, r)

	if err != nil {
		t.Error("unexpected error:", err)
	}

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
	handler := newHandler(st.NewStubClient().WithRoles(st.GetRoleResult{"", fmt.Errorf("unexpected error")}, st.GetRoleResult{"foo_role", nil}))

	status, err := handler.Handle(context.Background(), rr, r)

	if err != nil {
		t.Error(err)
	}

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
	handler := newHandler(st.NewStubClient().WithRoles(st.GetRoleResult{"", nil}))

	status, err := handler.Handle(context.Background(), rr, r)

	if status != http.StatusNotFound {
		t.Error("expected 404 response, was", status)
	}

	if err != EmptyRoleError {
		t.Error("unexpected error, was", err)
	}
}

func TestReturnErrorWhenPodNotFoundWithinTimeout(t *testing.T) {
	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/", nil)
	rr := httptest.NewRecorder()
	handler := newHandler(st.NewStubClient().WithRoles(st.GetRoleResult{"", server.ErrPodNotFound}))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	status, err := handler.Handle(ctx, rr, r)

	if status != http.StatusInternalServerError {
		t.Error("expected internal server error, was:", status)
	}
	if err != server.ErrPodNotFound {
		t.Error("unexpected error, was", err)
	}
}

func newHandler(c server.Client) *roleHandler {
	ip := func(r *http.Request) (string, error) {
		return "", nil
	}

	return &roleHandler{
		client:   c,
		clientIP: ip,
	}
}
