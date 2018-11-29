package metadata

import (
	"context"
	"fmt"
	"github.com/fortytw2/leaktest"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/uswitch/kiam/pkg/server"
	st "github.com/uswitch/kiam/pkg/testutil/server"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRedirectsToCanonicalPath(t *testing.T) {
	defer leaktest.Check(t)()

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

func readPrometheusCounterValue(name, labelName, labelValue string) float64 {
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		panic(err)
	}
	for _, m := range metrics {
		if m.GetName() == name {
			for _, metric := range m.Metric {
				for _, label := range metric.Label {
					if label.GetName() == labelName && label.GetValue() == labelValue {
						return metric.Counter.GetValue()
					}
				}
			}
		}
	}
	return 0
}

func TestIncrementsPrometheusCounter(t *testing.T) {
	defer leaktest.Check(t)()

	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/", nil)
	rr := httptest.NewRecorder()

	handler := newRoleHandler(st.NewStubClient().WithRoles(st.GetRoleResult{"foo_role", nil}), getBlankClientIP)
	router := mux.NewRouter()
	handler.Install(router)

	router.ServeHTTP(rr, r)

	responses := readPrometheusCounterValue("kiam_metadata_responses_total", "handler", "roleName")
	if responses != 1 {
		t.Error("expected responses_total to be 1, was", responses)
	}
	successes := readPrometheusCounterValue("kiam_metadata_success_total", "handler", "roleName")
	if successes != 1 {
		t.Error("expected success_total to be 1, was", successes)
	}
}

func TestReturnRoleWhenClientResponds(t *testing.T) {
	defer leaktest.Check(t)()

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
	defer leaktest.Check(t)()

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
	defer leaktest.Check(t)()

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
	defer leaktest.Check(t)()

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
