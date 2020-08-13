# Pipeline Testing

This module tests the full pipeline. Metrics data is collected from `pipeline.go` and sent
through the Exporter to Cortex.

## Setting up the Pipeline

### Cortex
The pipeline test requires a Cortex instance to be running since the Exporter is sending
POST requests to Cortex's endpoint. The Cortex instance runs using one of its demo
configuration files for a single process.

Install Instructions:
```shell
git clone git@github.com:cortexproject/cortex.git
go build ./cmd/cortex
./cortex -config.file=./docs/configuration/single-process-config.yaml
```

### Grafana
The demo verifies that the export and remote_write succeeded by checking the data on
Grafana.

Install Instructions:
```shell
docker run --rm -d --name=grafana -p 3000:3000 grafana/grafana
```

Afterwards, 
1. Go to `localhost:3000`
2. Login with `admin:admin` for `username:password`
3. Add `http://host.docker.internal:9009/api/prom` as a Prometheus data source
4. Add a dashboard and query either `a_counter` or `a_value_recorder`

### Running Pipeline
To run the pipeline test, change directory to the `pipeline` folder and run `go test`.
Make sure Cortex and Grafana are running beforehand. After all 3 services are running,
results should show up on Grafana.