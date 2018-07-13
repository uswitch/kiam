package metadata

import (
	"net/http"
	"testing"
)

func TestParseAddress(t *testing.T) {
	ip, err := ParseClientIP("127.0.0.1:9000")
	if err != nil {
		t.Fatal(err.Error())
	}

	if ip != "127.0.0.1" {
		t.Error("incorrect ip, was", ip)
	}
}

func getBlankClientIP(_ *http.Request) (string, error) {
	return "", nil
}
