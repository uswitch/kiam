package k8s

import (
	"errors"
	"fmt"
	"strings"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	auth "k8s.io/api/authentication/v1"
	"k8s.io/client-go/kubernetes"
)

type TokenReviewer struct {
	Client    kubernetes.Interface
	Namespace string
	Name      string
}

type tokenReviewResult struct {
	Name      string
	Namespace string
	UID       string
}

type KiamReview struct {
	*TokenReviewer
	*auth.TokenReview
}

func NewTokenReviewer(client *kubernetes.Clientset, namespace, serviceAccountName string) *TokenReviewer {
	return &TokenReviewer{Client: client, Namespace: namespace, Name: serviceAccountName}
}

func (t *TokenReviewer) CreateReview(token string) (*KiamReview, error) {
	review, err := t.Client.AuthenticationV1().TokenReviews().Create(
		&auth.TokenReview{Spec: auth.TokenReviewSpec{Token: token}})
	kiamReview := &KiamReview{TokenReviewer: t, TokenReview: review}
	return kiamReview, err
}

func (k *KiamReview) Review() (bool, error) {
	tokenReviewTime := metrics.GetOrRegisterTimer("tokenReview", metrics.DefaultRegistry)
	startTime := time.Now()
	defer tokenReviewTime.UpdateSince(startTime)

	if !k.TokenReview.Status.Authenticated {
		return false, fmt.Errorf("invalid token %s\n", k.TokenReview.Spec.Token)
	}
	log.Debugf("Token Review: [ %v ]", k.TokenReview)

	return k.validateUserInfo(k.Status.User)
}

func (k *KiamReview) validateUserInfo(user auth.UserInfo) (bool, error) {
	parts := strings.Split(user.Username, ":")
	if len(parts) != 4 {
		return false, errors.New("lookup failed: unexpected username format")
	}

	// Validate the user that comes back from token review is a
	// service account
	if parts[0] != "system" || parts[1] != "serviceaccount" {
		return false, errors.New("lookup failed: username returned is not a service account")
	}
	name := parts[3]
	namespace := parts[2]
	uid := string(user.UID)
	log.WithFields(log.Fields{
		"name":      name,
		"namespace": namespace,
		"uid":       uid,
	}).Info("token review results")
	valid := true
	if k.Namespace != "" && k.Name != "" {
		log.Debugf("checking name match")
		valid = k.Namespace == namespace && k.Name == name
	}
	if !valid {
		return false, fmt.Errorf("token %s:%s did not match %s:%s", namespace, name, k.Namespace, k.Name)
	}
	return valid, nil
}
