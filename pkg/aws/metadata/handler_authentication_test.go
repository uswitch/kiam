package metadata

import (
 	"context"
	"github.com/fortytw2/leaktest"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type testHandler struct {
	Status int
	Err error
}

func (h testHandler) Handle(_ context.Context, res http.ResponseWriter, _ *http.Request) (int, error) {
	res.Write([]byte("handling"))
	return h.Status, h.Err
}

func installAsTestHandler(h handler, router *mux.Router) {
	handler := adapt(withMeter("test", h))
	router.Handle("/test", handler)
}

func TestGoodAuth(t *testing.T) {
	defer leaktest.Check(t)()

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("unused"))
	}))
	defer func() { testServer.Close() }()

	r, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Error("Error creating http request")
	}
	rr := httptest.NewRecorder()
	rootHandler := testHandler{Status: http.StatusOK, Err: nil}

	authUrl, err  := url.Parse(testServer.URL)
	if err != nil {
		t.Error("Error getting server URL")
	}

	handler := NewAuthenticatingHandler(rootHandler, *authUrl)

	router := mux.NewRouter()
	installAsTestHandler(handler, router)

	router.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Error("Expected 200 response, was", rr.Code)
	}

	body, err := ioutil.ReadAll(rr.Body)
	if err != nil {
		t.Error("Error reading body of metadata response")
	}

	if string(body) != "handling" {
		t.Errorf("Did not receive expected response body. Got: %s", string(body))
	}
}

func TestBadAuth(t *testing.T) {
	defer leaktest.Check(t)()

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusForbidden)
		res.Write([]byte("nope"))
	}))
	defer func() { testServer.Close() }()

	r, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Error("Error creating http request")
	}
	rr := httptest.NewRecorder()
	rootHandler := testHandler{Status: http.StatusOK, Err: nil}

	authUrl, err  := url.Parse(testServer.URL)
	if err != nil {
		t.Error("Error getting server URL")
	}

	handler := NewAuthenticatingHandler(rootHandler, *authUrl)

	router := mux.NewRouter()
	installAsTestHandler(handler, router)

	router.ServeHTTP(rr, r)
	if rr.Code != http.StatusForbidden {
		t.Error("Expected 403 response, was", rr.Code)
	}

	body, err := ioutil.ReadAll(rr.Body)
	if err != nil {
		t.Error("Error reading body of metadata response")
	}

	if string(body) != "nope" {
		t.Errorf("Did not receive expected response body. Got: %s", string(body))
	}
}

func TestHeadersArePassedThrough(t *testing.T) {
	defer leaktest.Check(t)()

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if(len(req.Header["X-Aws-Ec2-Metadata-Token"]) > 0) {
			res.WriteHeader(http.StatusOK)
			res.Write([]byte("yup"))
		} else {
			res.WriteHeader(http.StatusForbidden)
			res.Write([]byte("nope"))
		}
	}))
	defer func() { testServer.Close() }()

	r, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Error("Error creating http request")
	}

	r.Header.Add("X-aws-ec2-metadata-token", "a token")
	rr := httptest.NewRecorder()
	rootHandler := testHandler{Status: http.StatusOK, Err: nil}

	authUrl, err  := url.Parse(testServer.URL)
	if err != nil {
		t.Error("Error getting server URL")
	}

	handler := NewAuthenticatingHandler(rootHandler, *authUrl)

	router := mux.NewRouter()
	installAsTestHandler(handler, router)

	router.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Error("Expected the X-Aws-Ec2-Metadata-Token header to be passed through, but got a non-200 response indicating that this is not the case. Response code: ", rr.Code)
	}
}
