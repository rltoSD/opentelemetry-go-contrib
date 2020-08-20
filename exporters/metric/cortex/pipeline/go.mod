module go.opentelemetry.io/contrib/exporters/metric/cortex/pipeline

go 1.14

replace go.opentelemetry.io/contrib/exporters/metric/cortex => ../cortex/

require (
	go.opentelemetry.io/contrib/exporters/metric/cortex v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/contrib/exporters/metric/cortex/utils v0.0.0-20200813041938-b948cd370862
	go.opentelemetry.io/otel v0.10.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.10.0
	go.opentelemetry.io/otel/sdk v0.10.0
)
