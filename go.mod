module github.com/grafana/k6build

go 1.22.2

require (
	github.com/grafana/clireadme v0.1.0
	github.com/grafana/k6catalog v0.2.4
	github.com/grafana/k6foundry v0.3.0
	github.com/spf13/cobra v1.8.1
)

require github.com/Masterminds/semver/v3 v3.3.1 // indirect

require (
	github.com/google/go-cmp v0.6.0
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/mod v0.21.0 // indirect
)

retract v0.0.0 // premature publishing
