module github.com/cyoung/stratux

go 1.14

require (
	github.com/cyoung/goflying v0.0.0-20190924175116-d0268b1b182f
	github.com/dustin/go-humanize v1.0.0
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/gansidui/geohash v0.0.0-20141019080235-ebe5ba447f34
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/jpoirier/gortlsdr v2.10.0+incompatible
	github.com/kellydunn/golang-geo v0.7.0
	github.com/kidoman/embd v0.0.0-20170508013040-d3d8c0c5c68d
	github.com/kylelemons/go-gypsy v0.0.0-20160905020020-08cad365cd28 // indirect
	github.com/lib/pq v1.3.0 // indirect
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/prometheus/client_golang v1.5.1
	github.com/ricochet2200/go-disk-usage v0.0.0-20150921141558-f0d1b743428f
	github.com/skelterjohn/go.matrix v0.0.0-20130517144113-daa59528eefd // indirect
	github.com/stratux/serial v0.0.0-19700101022104-87f23b1d3198
	github.com/takama/daemon v0.11.0
	github.com/tarm/serial v0.0.0-20180830185346-98f6abe2eb07
	github.com/uavionix/serial v0.0.0-19700101022104-87f23b1d3198
	github.com/ziutek/mymysql v1.5.4 // indirect
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a
	gonum.org/v1/plot v0.7.0
)

// Substitute until goflying is updated with module support
// replace github.com/cyoung/goflying => ../goflying
