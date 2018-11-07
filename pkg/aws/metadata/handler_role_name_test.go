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

func TestRedirectsToCanonicalPath(t *testing.T) {
	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials", nil)
	rr := httptest.NewRecorder()

	handler := newRoleHandler(nil, nil)
	router := mux.NewRouter()
	handler.Install(router)

	router.ServeHTTP(rr, r)

	if rr.Code != http.StatusMovedPermanently {
		t.Error("expected redirect, was", rr.Code)
	}
}

func TestReturnRoleWhenClientResponds(t *testing.T) {
	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/", nil)
	rr := httptest.NewRecorder()
	handler := newRoleHandler(st.NewStubClient().WithRoles(st.GetRoleResult{"foo_role", nil}), getBlankClientIP)
	router := mux.NewRouter()
	handler.Install(router)

	router.ServeHTTP(rr, r)

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
	handler := newRoleHandler(st.NewStubClient().WithRoles(st.GetRoleResult{"", fmt.Errorf("unexpected error")}, st.GetRoleResult{"foo_role", nil}), getBlankClientIP)
	router := mux.NewRouter()
	handler.Install(router)

	router.ServeHTTP(rr, r)

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
	handler := newRoleHandler(st.NewStubClient().WithRoles(st.GetRoleResult{"", nil}), getBlankClientIP)
	router := mux.NewRouter()
	handler.Install(router)

	router.ServeHTTP(rr, r)

	if rr.Code != http.StatusNotFound {
		t.Error("expected 404 response, was", rr.Code)
	}
}

func TestReturnErrorWhenPodNotFoundWithinTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/", nil)
	rr := httptest.NewRecorder()
	handler := newRoleHandler(st.NewStubClient().WithRoles(st.GetRoleResult{"", server.ErrPodNotFound}), getBlankClientIP)
	router := mux.NewRouter()
	handler.Install(router)

	router.ServeHTTP(rr, r.WithContext(ctx))

	if rr.Code != http.StatusInternalServerError {
		t.Error("expected internal server error, was:", rr.Code)
	}
}
