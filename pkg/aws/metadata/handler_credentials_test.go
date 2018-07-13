package metadata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/server"
	st "github.com/uswitch/kiam/pkg/testutil/server"
)

func TestReturnsCredentials(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/role", nil)
	rr := httptest.NewRecorder()

	client := st.NewStubClient().WithRoles(st.GetRoleResult{"role", nil}).WithCredentials(st.GetCredentialsResult{&sts.Credentials{AccessKeyId: "A1", SecretAccessKey: "S1"}, nil})
	handler := newCredentialsHandler(client, blankIPResolver)
	router := mux.NewRouter()
	handler.Install(router)

	router.ServeHTTP(rr, r.WithContext(ctx))

	if rr.Code != http.StatusOK {
		t.Error("unexpected status, was", rr.Code)
	}

	content := rr.Header().Get("Content-Type")
	if content != "application/json" {
		t.Error("expected json result", content)
	}

	var creds sts.Credentials
	decoder := json.NewDecoder(rr.Body)
	err := decoder.Decode(&creds)
	if err != nil {
		t.Error(err.Error())
	}

	if creds.AccessKeyId != "A1" {
		t.Error("unexpected key, was", creds.AccessKeyId)
	}
	if creds.SecretAccessKey != "S1" {
		t.Error("unexpected secret key, was", creds.SecretAccessKey)
	}
}

func TestReturnsErrorWithNoPod(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	r, _ := http.NewRequest("GET", "/latest/meta-data/iam/security-credentials/role", nil)
	rr := httptest.NewRecorder()

	client := st.NewStubClient().WithCredentials(st.GetCredentialsResult{nil, server.ErrPodNotFound})
	handler := newCredentialsHandler(client, blankIPResolver)
	router := mux.NewRouter()
	handler.Install(router)

	router.ServeHTTP(rr, r.WithContext(ctx))

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
	handler := newCredentialsHandler(client, blankIPResolver)
	router := mux.NewRouter()
	handler.Install(router)

	router.ServeHTTP(rr, r.WithContext(ctx))

	if rr.Code != http.StatusOK {
		t.Error("unexpected status", rr.Code)
	}
}
