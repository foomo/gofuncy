module github.com/foomo/gofuncy/examples

go 1.24.2

replace github.com/foomo/gofuncy => ../

require (
	github.com/foomo/gofuncy v0.0.0-00010101000000-000000000000
	github.com/uptrace/opentelemetry-go-extra/otelzap v0.3.2
	go.opentelemetry.io/otel v1.35.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.11.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.35.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.35.0
	go.opentelemetry.io/otel/log v0.11.0
	go.opentelemetry.io/otel/sdk v1.35.0
	go.opentelemetry.io/otel/sdk/log v0.11.0
	go.opentelemetry.io/otel/sdk/metric v1.35.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/Ju0x/humanhash v1.0.2 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/uptrace/opentelemetry-go-extra/otelutil v0.3.2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
)
