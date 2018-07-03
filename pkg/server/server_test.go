package server

import (
	"context"
	"github.com/uswitch/kiam/pkg/k8s"
	pb "github.com/uswitch/kiam/proto"
	kt "k8s.io/client-go/tools/cache/testing"
	"testing"
	"time"
)

const (
	defaultBuffer = 10
)

func TestReturnsErrorWhenPodNotFound(t *testing.T) {
	source := kt.NewFakeControllerSource()
	podCache := k8s.NewPodCache(source, time.Second, defaultBuffer)
	server := &KiamServer{pods: podCache}

	_, err := server.GetPodCredentials(context.Background(), &pb.GetPodCredentialsRequest{})

	if err != ErrPodNotFound {
		t.Error("unexpected error:", err)
	}
}
