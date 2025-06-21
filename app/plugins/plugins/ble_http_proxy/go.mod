module github.com/NetScout-Go/Plugin_ble_http_proxy

go 1.24.4

require (
	github.com/NetScout-Go/NetTool v0.0.0
	github.com/godbus/dbus/v5 v5.1.0
	github.com/muka/go-bluetooth v0.0.0-20221213043340-85dc80edc4e1
)

require (
	github.com/fatih/structs v1.1.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	golang.org/x/sys v0.33.0 // indirect
)

replace github.com/NetScout-Go/NetTool => ../../../..
