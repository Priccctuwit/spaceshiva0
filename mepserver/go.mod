module mepserver

go 1.14

replace (
	github.com/coreos/etcd v3.3.6+incompatible => github.com/coreos/etcd v3.3.13+incompatible
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b => github.com/go-chassis/glog v0.0.0-20180920075250-95a09b2413e9
	github.com/gorilla/websocket v1.2.0 => github.com/gorilla/websocket v1.4.1
	k8s.io/client-go v2.0.0-alpha.0.0.20180817174322-745ca8300397+incompatible => github.com/kubernetes/client-go v0.0.0-20180817174322-745ca8300397
)

require (
	github.com/agiledragon/gomonkey v2.0.1+incompatible
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/apache/servicecomb-service-center v0.0.0-20191027084911-c2dc0caef706
	github.com/go-chassis/paas-lager v1.1.1 // indirect
	github.com/go-mesh/openlogging v1.0.1 // indirect
	github.com/go-playground/validator/v10 v10.2.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0 // indirect
	github.com/prometheus/common v0.0.0-20180801064454-c7de2306084e
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.4.0
	golang.org/x/crypto v0.0.0-20200302210943-78000ba7a073
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a
	gopkg.in/alecthomas/kingpin.v2 v2.2.6 // indirect
)
