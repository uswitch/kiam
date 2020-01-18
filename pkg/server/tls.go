package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	clientTLSMetrics = newTLSMetrics("client")
	serverTLSMetrics = newTLSMetrics("server")
)

type tlsMetrics struct {
	registerOnce sync.Once
	verifyError  prometheus.Gauge
	expiration   prometheus.Gauge
}

func newTLSMetrics(subsystem string) *tlsMetrics {
	return &tlsMetrics{
		verifyError: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "grpc",
			Subsystem: subsystem,
			Name:      "tls_certificate_verify_error",
			Help:      "Indicates if there was an error verifying the latest gRPC " + subsystem + " TLS certificate and its expiration.",
		}),
		expiration: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "grpc",
			Subsystem: subsystem,
			Name:      "tls_certificate_expiry_seconds",
			Help:      "Expiration time of the gRPC " + subsystem + " TLS certificate in seconds since the Unix epoch.",
		}),
	}
}

func (m *tlsMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.verifyError.Describe(ch)
	m.expiration.Describe(ch)
}

func (m *tlsMetrics) Collect(ch chan<- prometheus.Metric) {
	m.verifyError.Collect(ch)
	m.expiration.Collect(ch)
}

func (m *tlsMetrics) update(usage x509.ExtKeyUsage, cert *tls.Certificate, pool *x509.CertPool) {
	m.registerOnce.Do(func() { prometheus.MustRegister(m) })

	expiry, err := earliestExpiry(cert, pool, usage)
	if err != nil {
		log.Errorf("failed to verify TLS certificate: %v", err)
		m.verifyError.Set(1)
	} else {
		m.verifyError.Set(0)
	}
	m.expiration.Set(float64(expiry.Unix()))
}

func loadCerts(certFile, keyFile, caFile string) (tls.Certificate, *x509.CertPool, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("error loading TLS key pair: %v", err)
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("error parsing TLS leaf cert: %v", err)
	}
	ca, err := ioutil.ReadFile(caFile)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("error reading TLS CAs: %v", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(ca) {
		return tls.Certificate{}, nil, fmt.Errorf("error parsing TLS CAs")
	}
	return cert, caPool, nil
}

func earliestExpiry(cert *tls.Certificate, pool *x509.CertPool, usage x509.ExtKeyUsage) (time.Time, error) {
	x509Cert := cert.Leaf
	if x509Cert == nil {
		var err error
		if x509Cert, err = x509.ParseCertificate(cert.Certificate[0]); err != nil {
			return time.Time{}, err
		}
	}
	chains, err := x509Cert.Verify(x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{usage},
	})
	if err != nil {
		return x509Cert.NotAfter, err
	}
	var t time.Time
	for _, chain := range chains {
		for _, cert := range chain {
			if t.IsZero() || cert.NotAfter.Before(t) {
				t = cert.NotAfter
			}
		}
	}
	return t, nil
}
