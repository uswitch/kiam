package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/fsnotify.v1"
)

var (
	clientTLSMetrics = newTLSMetrics("client")
	serverTLSMetrics = newTLSMetrics("server")
)

type tlsCertsNotifyFunc func(*tls.Certificate, *x509.CertPool, error)

type tlsMetrics struct {
	registerOnce sync.Once
	updateError  prometheus.Gauge
	verifyError  prometheus.Gauge
	expiration   prometheus.Gauge
}

func newTLSMetrics(subsystem string) *tlsMetrics {
	return &tlsMetrics{
		updateError: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "grpc",
			Subsystem: subsystem,
			Name:      "tls_certificate_update_error",
			Help:      "Indicates if there was an error updating the latest gRPC " + subsystem + " TLS certificate.",
		}),
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
	m.updateError.Describe(ch)
	m.verifyError.Describe(ch)
	m.expiration.Describe(ch)
}

func (m *tlsMetrics) Collect(ch chan<- prometheus.Metric) {
	m.updateError.Collect(ch)
	m.verifyError.Collect(ch)
	m.expiration.Collect(ch)
}

func (m *tlsMetrics) notifyFunc(usage x509.ExtKeyUsage) tlsCertsNotifyFunc {
	m.registerOnce.Do(func() { prometheus.MustRegister(m) })
	return func(cert *tls.Certificate, pool *x509.CertPool, err error) {
		if err != nil {
			log.Errorf("failed to update TLS certificate: %v", err)
			m.updateError.Set(1)
			return
		}
		m.updateError.Set(0)

		expiry, err := earliestExpiry(cert, pool, usage)
		if err != nil {
			log.Errorf("failed to verify TLS certificate: %v", err)
			m.verifyError.Set(1)
		} else {
			m.verifyError.Set(0)
		}
		m.expiration.Set(float64(expiry.Unix()))
	}
}

type tlsCerts struct {
	cert *tls.Certificate
	pool *x509.CertPool
}

const hashSize = 16 // 128-bit

type dynamicTLSConfig struct {
	latest atomic.Value
	hash   [hashSize]byte // dedupes notify calls

	certFile string
	keyFile  string
	caFile   string
	notifyFn tlsCertsNotifyFunc

	close   sync.Once         // protects watcher from multiple calls to Close
	watcher *fsnotify.Watcher // watches directories containing files
	done    chan struct{}     // signals end of watch goroutine
}

func newDynamicTLSConfig(certFile, keyFile, caFile string, notifyFn tlsCertsNotifyFunc) (cfg *dynamicTLSConfig, err error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			w.Close()
		}
	}()
	if notifyFn == nil {
		notifyFn = func(*tls.Certificate, *x509.CertPool, error) {}
	}
	cfg = &dynamicTLSConfig{
		certFile: filepath.Clean(certFile),
		keyFile:  filepath.Clean(keyFile),
		caFile:   filepath.Clean(caFile),
		notifyFn: notifyFn,
		watcher:  w,
		done:     make(chan struct{}),
	}
	for dir := range map[string]bool{
		filepath.Dir(cfg.certFile): true,
		filepath.Dir(cfg.keyFile):  true,
		filepath.Dir(cfg.caFile):   true,
	} {
		if err := w.Add(dir); err != nil {
			return nil, err
		}
	}
	if err := cfg.read(); err != nil {
		return nil, err
	}
	go cfg.watch()
	return cfg, nil
}

func (cfg *dynamicTLSConfig) Close() error {
	cfg.close.Do(func() { cfg.watcher.Close() })
	<-cfg.done
	return nil
}

func (cfg *dynamicTLSConfig) Load() (*tls.Certificate, *x509.CertPool) {
	v := cfg.latest.Load().(*tlsCerts)
	return v.cert, v.pool
}

func (cfg *dynamicTLSConfig) LoadCert() *tls.Certificate {
	cert, _ := cfg.Load()
	return cert
}

func (cfg *dynamicTLSConfig) LoadCACerts() *x509.CertPool {
	_, pool := cfg.Load()
	return pool
}

// read reads the files from disk, parses them, and returns any error.
// If there is a change in certs, it stores them and calls notifyFn.
// All calls to read occur serailly, once from the constructor and
// then from the watch goroutine as fsnotify events are processed.
func (cfg *dynamicTLSConfig) read() error {
	certPEMBlock, err := ioutil.ReadFile(cfg.certFile)
	if err != nil {
		return fmt.Errorf("error reading TLS cert: %v", err)
	}
	keyPEMBlock, err := ioutil.ReadFile(cfg.keyFile)
	if err != nil {
		return fmt.Errorf("error reading TLS key: %v", err)
	}
	caPEMCerts, err := ioutil.ReadFile(cfg.caFile)
	if err != nil {
		return fmt.Errorf("error reading TLS CAs: %v", err)
	}
	// hash to dedupe notifications
	var sum [hashSize]byte
	h := fnv.New128a()
	h.Write(certPEMBlock)
	h.Write(keyPEMBlock)
	h.Write(caPEMCerts)
	h.Sum(sum[:0])
	if cfg.hash == sum {
		return nil
	}
	cfg.hash = sum

	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err != nil {
		return fmt.Errorf("error parsing TLS keypair: %v", err)
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("error parsing TLS leaf cert: %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEMCerts) {
		return fmt.Errorf("error parsing TLS CAs")
	}

	cfg.latest.Store(&tlsCerts{&cert, pool})
	cfg.notifyFn(&cert, pool, nil)
	return nil
}

func (cfg *dynamicTLSConfig) watch() {
	defer close(cfg.done)
	for {
		select {
		case _, ok := <-cfg.watcher.Events:
			if !ok {
				return
			}
			// TODO: ignore unrelated events
			if err := cfg.read(); err != nil {
				log.Errorf("tls config read error: %v", err)
				cfg.notifyFn(nil, nil, err)
			}
		case err, ok := <-cfg.watcher.Errors:
			if !ok {
				return
			}
			log.Errorf("tls config watch error: %v", err)
		}
	}
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
