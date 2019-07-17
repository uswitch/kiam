package metadata

import (
	"github.com/fortytw2/leaktest"
	"github.com/gorilla/mux"
	st "github.com/uswitch/kiam/pkg/testutil/server"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthReturn(t *testing.T) {
	defer leaktest.Check(t)()
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("i-12345"))
	}))
	defer func() { testServer.Close() }()

	r, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Error("Error creating http request")
	}
	rr := httptest.NewRecorder()
	handler := newHealthHandler(st.NewStubClient(), testServer.URL)
	router := mux.NewRouter()
	handler.Install(router)
	router.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Error("expected 200 response, was", rr.Code)
	}
	body, err := ioutil.ReadAll(rr.Body)
	if err != nil {
		t.Error("error reading body of metadata response")
	}
	if string(body) != "i-12345" {
		t.Error("instance-id not returned correctly")
	}
}

func TestDeepHealthBadReturn(t *testing.T) {
	defer leaktest.Check(t)()
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("i-12345"))
	}))
	defer func() { testServer.Close() }()

	r, err := http.NewRequest("GET", "/health?deep=true", nil)
	if err != nil {
		t.Error("Error creating http request")
	}
	rr := httptest.NewRecorder()
	handler := newHealthHandler(st.NewStubClient().WithHealth("bad"), testServer.URL)
	router := mux.NewRouter()
	handler.Install(router)
	router.ServeHTTP(rr, r)
	if rr.Code != http.StatusInternalServerError {
		t.Error("expected 500 response, was", rr.Code)
	}
}

func TestDeepHealthReturn(t *testing.T) {
	defer leaktest.Check(t)()
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("i-12345"))
	}))
	defer func() { testServer.Close() }()

	r, err := http.NewRequest("GET", "/health?deep=true", nil)
	if err != nil {
		t.Error("Error creating http request")
	}
	rr := httptest.NewRecorder()
	handler := newHealthHandler(st.NewStubClient().WithHealth("ok"), testServer.URL)
	router := mux.NewRouter()
	handler.Install(router)
	router.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Error("expected 200 response, was", rr.Code)
	}
	body, err := ioutil.ReadAll(rr.Body)
	if err != nil {
		t.Error("error reading body of metadata response")
	}
	if string(body) != "i-12345" {
		t.Error("instance-id not returned correctly")
	}
}
