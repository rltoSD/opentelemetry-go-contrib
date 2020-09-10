module go.opentelemetry.io/contrib/exporters/metric/cortex/pipeline

go 1.14

// replace go.opentelemetry.io/contrib/exporters/metric/cortex => ../cortex/
replace go.opentelemetry.io/contrib/exporters/metric/cortex => ../

require (
	github.com/beorn7/perks v1.0.1
	github.com/cheggaaa/pb/v3 v3.0.5
	github.com/sergi/go-diff v1.1.0
	github.com/stretchr/testify v1.6.1
	github.com/tidwall/gjson v1.6.0
	go.opentelemetry.io/contrib/exporters/metric/cortex v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/contrib/exporters/metric/cortex/utils v0.0.0-20200813041938-b948cd370862
	go.opentelemetry.io/otel v0.11.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.11.0
	go.opentelemetry.io/otel/sdk v0.11.0
	go.uber.org/zap v1.10.0
)
