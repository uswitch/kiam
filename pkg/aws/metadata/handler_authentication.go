package metadata

import (
	"bytes"
 	"context"
	"net/http"
	"net/url"
)

type authenticatingHandler struct {
	downstreamHandler handler
	client            http.Client
	endpoint          url.URL
}

func NewAuthenticatingHandler(downstreamHandler handler, metadataEndpoint url.URL) authenticatingHandler {
	return authenticatingHandler{
		downstreamHandler: downstreamHandler,
		endpoint: metadataEndpoint,
	}
}

func (h authenticatingHandler) Handle(ctx context.Context, w http.ResponseWriter, req *http.Request) (int, error) {
	authURL := h.endpoint
	authURL.Path = req.URL.Path

	authContext := context.Background()

	authRequest  := req.Clone(authContext)
	authRequest.URL = &authURL

	authResponse, err := h.client.Do(authRequest)

	if err == nil && authResponse.StatusCode == http.StatusOK {
		return h.downstreamHandler.Handle(ctx, w, req)
	} else if err != nil {
		return http.StatusBadGateway, err
	} else {
		buffer := bytes.NewBuffer(make([]byte, 0, authResponse.ContentLength))
		_, err = buffer.ReadFrom(authResponse.Body)

		w.WriteHeader(authResponse.StatusCode)
		w.Write(buffer.Bytes())
		return authResponse.StatusCode, nil
	}
}
