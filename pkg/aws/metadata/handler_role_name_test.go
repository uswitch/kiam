package metadata

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/uswitch/kiam/pkg/server"
	st "github.com/uswitch/kiam/pkg/testutil/server"
)

func TestReturnRoleWhenClientResponds(t *testing.T) {
	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/", nil)
	rr := httptest.NewRecorder()
	handler := newHandler(st.NewStubClient().WithRoles(st.GetRoleResult{"foo_role", nil}))

	handler.ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Error("expected 200 response, was", rr.Code)
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

	handler.ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Error("expected 200 response, was", rr.Code)
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

	handler.ServeHTTP(rr, r)

	if rr.Code != http.StatusNotFound {
		t.Error("expected 404 response, was", rr.Code)
	}
}

func TestReturnErrorWhenPodNotFoundWithinTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/", nil)
	rr := httptest.NewRecorder()
	handler := newHandler(st.NewStubClient().WithRoles(st.GetRoleResult{"", server.ErrPodNotFound}))

	handler.ServeHTTP(rr, r.WithContext(ctx))

	if rr.Code != http.StatusInternalServerError {
		t.Error("expected internal server error, was:", rr.Code)
	}
}

func newHandler(c server.Client) http.Handler {
	ip := func(r *http.Request) (string, error) {
		return "", nil
	}

	h := &roleHandler{
		client:   c,
		clientIP: ip,
	}
	r := mux.NewRouter()
	h.Install(r)
	return r
}
