module github.com/textolytics/nbgo

go 1.22.3

require (
	github.com/textolytics/nbgo/cli v0.0.0
	github.com/textolytics/nbgo/conf v0.0.0
	github.com/textolytics/nbgo/core v0.0.0
	github.com/textolytics/nbgo/dw v0.0.0-20260121013637-a2a097c238fd
	github.com/textolytics/nbgo/gw v0.0.0-20260121013637-a2a097c238fd
	github.com/textolytics/nbgo/http v0.0.0
	github.com/textolytics/nbgo/logs v0.0.0
	github.com/textolytics/nbgo/mb v0.0.0-20260121013637-a2a097c238fd
	github.com/textolytics/nbgo/mon v0.0.0-20260121013637-a2a097c238fd
	github.com/textolytics/nbgo/run v0.0.0-20260121013637-a2a097c238fd
	github.com/textolytics/nbgo/task v0.0.0-20260121013637-a2a097c238fd
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gateio/gatews/go v0.0.0-20250523113507-90357b11b694 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/sys v0.12.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/textolytics/nbgo/cli => ./cli
	github.com/textolytics/nbgo/conf => ./conf
	github.com/textolytics/nbgo/core => ./core
	github.com/textolytics/nbgo/dw => ./dw
	github.com/textolytics/nbgo/gui => ./gui
	github.com/textolytics/nbgo/gw => ./gw
	github.com/textolytics/nbgo/http => ./http
	github.com/textolytics/nbgo/logs => ./logs
	github.com/textolytics/nbgo/mb => ./mb
	github.com/textolytics/nbgo/mcp => ./mcp
	github.com/textolytics/nbgo/mon => ./mon
	github.com/textolytics/nbgo/run => ./run
	github.com/textolytics/nbgo/task => ./task
)
