module github.com/medcl/esm

go 1.22

require (
	github.com/cheggaaa/pb v1.0.29
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575
	github.com/jessevdk/go-flags v1.5.0
	github.com/mattn/go-isatty v0.0.14
	github.com/parnurzeal/gorequest v0.2.16
	github.com/valyala/fasthttp v1.34.0
	infini.sh/framework v0.0.0
)

replace infini.sh/framework => ./framework
