module github.com/textolytics/nbgo/gui

go 1.22.3

require (
	github.com/textolytics/nbgo/cli v0.0.0
	github.com/textolytics/nbgo/conf v0.0.0
	github.com/textolytics/nbgo/core v0.0.0
	github.com/textolytics/nbgo/logs v0.0.0
)

require gopkg.in/yaml.v2 v2.4.0

replace (
	github.com/textolytics/nbgo/cli => ../cli
	github.com/textolytics/nbgo/conf => ../conf
	github.com/textolytics/nbgo/core => ../core
	github.com/textolytics/nbgo/logs => ../logs
)
