module github.com/overlaynetwork/onet-transport-mux-go

go 1.14

require (
	github.com/hashicorp/yamux v0.0.0-20200609203250-aecfd211c9ce // indirect
	github.com/inconshreveable/muxado v0.0.0-20160802230925-fc182d90f26e
	github.com/libs4go/errors v0.0.3
	github.com/libs4go/scf4go v0.0.8
	github.com/libs4go/slf4go v0.0.4
	github.com/mmcloughlin/avo v0.0.0-20200803215136-443f81d77104 // indirect
	github.com/overlaynetwork/onet-go v0.0.4
	github.com/overlaynetwork/onet-transport-kcp-go v0.0.0-20200914143241-31fabd85b0df
	github.com/stretchr/testify v1.6.1
	github.com/xtaci/lossyconn v0.0.0-20200209145036-adba10fffc37 // indirect
)

replace github.com/overlaynetwork/onet-go => ../onet-go

replace github.com/overlaynetwork/onet-transport-kcp-go => ../onet-transport-kcp-go
