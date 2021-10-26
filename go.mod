module github.com/uswitch/kiam

go 1.13

require (
	github.com/aws/aws-sdk-go v1.35.10
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/coreos/go-iptables v0.3.0
	github.com/fortytw2/leaktest v1.3.0
	github.com/golang/protobuf v1.4.3
	github.com/gorilla/mux v1.7.3
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.1-0.20190118093823-f849b5445de4
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/onsi/gomega v1.7.1 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/prometheus/client_golang v1.8.0
	github.com/sirupsen/logrus v1.6.0
	github.com/uswitch/k8sc v0.0.0-20170525133932-475c8175b340
	github.com/vmg/backoff v1.0.0
	golang.org/x/net v0.0.0-20201216054612-986b41b23924 // indirect
	golang.org/x/sys v0.0.0-20201214210602-f9fddec55a1e // indirect
	google.golang.org/genproto v0.0.0-20201214200347-8c77b98c765d // indirect
	google.golang.org/grpc v1.36.0
	google.golang.org/grpc/examples v0.0.0-20211020220737-f00baa6c3c84 // indirect
	google.golang.org/grpc/security/advancedtls v0.0.0-20200204204621-648cf9b00e25
	google.golang.org/protobuf v1.25.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/fsnotify.v1 v1.4.7
	k8s.io/api v0.20.0
	k8s.io/apimachinery v0.20.0
	k8s.io/client-go v0.20.0
)
