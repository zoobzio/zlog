module github.com/zoobzio/zlog/examples/event-pipeline

go 1.23.1

toolchain go1.24.5

require github.com/zoobzio/zlog v0.0.0

require github.com/zoobzio/pipz v0.6.0 // indirect

replace github.com/zoobzio/zlog => ../..

replace github.com/zoobzio/pipz => ../../../pipz
