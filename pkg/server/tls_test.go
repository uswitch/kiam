package server

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestDynamicTLS(t *testing.T) {
	// generate certs
	ca, caCertPEMBlock, _ := generateCert(t, nil)
	cert0, certPEMBlock0, keyPEMBlock0 := generateCert(t, ca)
	cert1, certPEMBlock1, keyPEMBlock1 := generateCert(t, ca)

	// See AtomicWriter for details of secret update algorithm used by kubelet:
	// https://godoc.org/k8s.io/kubernetes/pkg/volume/util#AtomicWriter.Write

	dir, err := ioutil.TempDir("", "")
	check(t, "Failed to create directory", err)
	defer os.RemoveAll(dir)

	// initialize data
	data := filepath.Join(dir, "..data")
	for _, name := range []string{"cert.pem", "key.pem", "roots.pem"} {
		check(t, "Failed to create symlink", os.Symlink(filepath.Join(data, name), filepath.Join(dir, name)))
	}
	data0 := filepath.Join(dir, "..data_0")
	createDir(t, data0, map[string][]byte{
		"cert.pem":  certPEMBlock0,
		"key.pem":   keyPEMBlock0,
		"roots.pem": caCertPEMBlock,
	})
	check(t, "Failed to create symlink", os.Symlink(data0, data))

	// create config
	certsCh := make(chan *tlsCerts, 1)
	wantCert := func(want *tls.Certificate) {
		t.Helper()
		select {
		case got := <-certsCh:
			if !reflect.DeepEqual(got.cert.Certificate, want.Certificate) {
				t.Fatal("Unexpected cert")
			}
			if !reflect.DeepEqual(got.cert.PrivateKey, want.PrivateKey) {
				t.Fatal("Unexpected key")
			}
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for certs")
		}
	}
	cfg, err := newDynamicTLSConfig(
		filepath.Join(dir, "cert.pem"),
		filepath.Join(dir, "key.pem"),
		filepath.Join(dir, "roots.pem"),
		func(cert *tls.Certificate, pool *x509.CertPool, err error) {
			select {
			case <-certsCh:
			default:
			}
			certsCh <- &tlsCerts{cert, pool}
		},
	)
	check(t, "Failed to initialize config", err)
	defer cfg.Close()
	wantCert(cert0)

	// update data
	data1 := filepath.Join(dir, "..data_1")
	createDir(t, data1, map[string][]byte{
		"cert.pem":  certPEMBlock1,
		"key.pem":   keyPEMBlock1,
		"roots.pem": caCertPEMBlock,
	})
	dataTmp := filepath.Join(dir, "..data_tmp")
	check(t, "Failed to create symlink", os.Symlink(data1, dataTmp))
	check(t, "Failed to rename symlink", os.Rename(dataTmp, data))
	wantCert(cert1)
}

func generateCert(t *testing.T, ca *tls.Certificate) (_ *tls.Certificate, certPEMBlock, keyPEMBlock []byte) {
	// See: https://golang.org/src/crypto/tls/generate_cert.go
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	check(t, "Failed to generate private key", err)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	check(t, "Failed to generate serial number", err)
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	var (
		parent *x509.Certificate
		key    crypto.Signer
	)
	if ca == nil { // self-signed
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
		parent = template
		key = priv
	} else {
		if parent = ca.Leaf; parent == nil {
			parent, err = x509.ParseCertificate(ca.Certificate[0])
			check(t, "Failed to parse CA certificate", err)
		}
		key = ca.PrivateKey.(crypto.Signer)
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, template, parent, &priv.PublicKey, key)
	check(t, "Failed to create certificate", err)
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	check(t, "Failed to marshal private key", err)
	certPEMBlock = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEMBlock = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	check(t, "Failed to unmarshal key pair", err)
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	check(t, "Failed to parse certificate", err)
	return &cert, certPEMBlock, keyPEMBlock
}

func createDir(t *testing.T, dir string, files map[string][]byte) {
	t.Helper()
	check(t, "Failed to make directory", os.Mkdir(dir, os.ModePerm))
	for name, buf := range files {
		check(t, "Failed to write file", ioutil.WriteFile(filepath.Join(dir, name), buf, os.ModePerm))
	}
}

func check(t *testing.T, msg string, err error) {
	if err != nil {
		t.Helper()
		t.Fatalf("%s: %v", msg, err)
	}
}
