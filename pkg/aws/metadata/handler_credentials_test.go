package metadata

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/server"
	st "github.com/uswitch/kiam/pkg/testutil/server"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReturnsCredentials(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/role", nil)
	rr := httptest.NewRecorder()

	client := st.NewStubClient().WithRoles(st.GetRoleResult{"role", nil}).WithCredentials(st.GetCredentialsResult{&sts.Credentials{AccessKeyId: "A1", SecretAccessKey: "S1"}, nil})
	handler := newCredentialsHandler(client)
	handler.ServeHTTP(rr, r.WithContext(ctx))

	if rr.Code != http.StatusOK {
		t.Error("unexpected status, was", rr.Code)
	}

	content := rr.Header().Get("Content-Type")
	if content != "application/json" {
		t.Error("expected json result", content)
	}

	expected := `{"Code":"","Type":"","AccessKeyId":"A1","SecretAccessKey":"S1","Token":"","Expiration":"","LastUpdated":""}`

	if !strings.Contains(rr.Body.String(), expected) {
		t.Error("unexpected result", rr.Body.String())
	}
}

func TestReturnsErrorWithNoPod(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/role", nil)
	rr := httptest.NewRecorder()

	client := st.NewStubClient().WithCredentials(st.GetCredentialsResult{nil, server.ErrPodNotFound})
	handler := newCredentialsHandler(client)
	handler.ServeHTTP(rr, r.WithContext(ctx))

	if rr.Code != http.StatusInternalServerError {
		t.Error("unexpected status", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "error fetching credentials: no pod found") {
		t.Error("unexpected error", rr.Body.String())
	}
}

func TestReturnsCredentialsWithRetryAfterError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/role", nil)
	rr := httptest.NewRecorder()

	valid := st.GetCredentialsResult{&sts.Credentials{}, nil}
	e := st.GetCredentialsResult{nil, server.ErrPodNotFound}
	client := st.NewStubClient().WithRoles(st.GetRoleResult{"role", nil}).WithCredentials(e, valid)
	handler := newCredentialsHandler(client)
	handler.ServeHTTP(rr, r.WithContext(ctx))

	if rr.Code != http.StatusOK {
		t.Error("unexpected status", rr.Code)
	}
}

func newCredentialsHandler(c server.Client) http.Handler {
	ip := func(r *http.Request) (string, error) {
		return "", nil
	}
	h := &credentialsHandler{clientIP: ip, client: c}
	r := mux.NewRouter()
	h.Install(r)
	return r
}
