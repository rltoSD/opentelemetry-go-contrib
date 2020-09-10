# Go Cortex Exporter Pipeline Test

This module implements an integration test for the Go Cortex Exporter. The test runs the
Exporter Setup pipeline to create a new push controller that periodically calls the Go
Cortex Exporter's `Export()` method. It then creates 12 instruments for each combination
of instrument and data type -- 6 instruments x 2 data types (int64 / float64). Test data
is read in from a generated CSV file line by line and used to add values to these
instruments. Note that generated data contains instruments that use unsupported data types
such as `short` and `double`. These lines will be interpreted as `int64` and `float64`
instruments. The test ends when the CSV file no longer has any lines to read. Since 

## Input Data
The input data is generated using a Jupyter notebook. It is a CSV file where each line
represents an instrument, the data to record, and the labels that should be added to said
instrument. An example line is:

`ictr,21,"name1, descr1, key3, value3, key4, value4, key2, value2, key1, value1"`

The first field is the instrument. The first letter represents the data type (int, float,
short, double, etc) and the rest of the field is instrument type. For example, `ictr`
means `int64 Counter`. The second field is the value to be recorded in the instrument. The
third field is a string that contains comma delimited strings. The first two substrings
are for the name and the description. Neither of them will be used in the pipeline test.
The strings afterwards represent key value pairs. Each group of two form a KeyValue.

All tested instruments are displayed in the table below:

| DataType/InstrumentType | Counter (`ctr`) | UpDownCounter (`udctr`) | ValueRecorder (`vrec`) | SumObserver (`sobs`) | UpDownObserver (`udobs`) | ValueObserver (`vobs`) |
|-------------------------|-----------------|-------------------------|------------------------|----------------------|--------------------------|------------------------|
| `int64` (`i`)           | `ictr`          | `iudctr`                | `ivrec`                | `isobs`              | `iudobs`                 | `ivobs`                |
| `float64` (`f`)         | `fctr`          | `fudctr`                | `fvrec`                | `fsobs`              | `fudobs`                 | `fvobs`                |

The Jupyter notebook generates four files with two files being reference data to use in the
pipeline test and two answer files to compare the results against.


## Verifying Results


## Running the Pipeline Test

This pipeline test consists of three components:
1. Cortex
2. Grafana
3. Go application (`pipeline.go`)

### Step 1 - Install and Start Cortex
The first step is to install and start an instance of Cortex. The Go Cortex Exporter's
purpose is to export metrics data to Cortex. Therefore, there must be a running instance
of Cortex to accept the exported data. Cortex can be installed and started with the shell
commands below. This pipeline test uses one of Cortex's example configuration files. This
configuration file has a ingestion limit of 25000, which is too low for larger data files.
Use the cortexConfig.yml file in repo to run larger tests.

```shell
git clone https://github.com/cortexproject/cortex.git
cd cortex
go build ./cmd/cortex

# For smaller tests. Default config file from documentation.
./cortex -config.file=./docs/configuration/single-process-config.yaml

# For larger tests. Very similar to default file above, but with larger limits.
./cortex -config.file=<cortexConfig.yml>
```

### Step 2 - Install, Start, and Setup Grafana
The second step is to install and start Grafana, an open-source analytics platform that
will be used to visualize the exported data in Cortex. Grafana offers an easy way to query
and graph the data and is a clear way to show that the exporter is working as intended.
This step is not required as the tests will still run; Grafana provides another way to
check the exported data.

Use the following shell command to install and run a docker image with Grafana.

```shell
docker run --rm -d --name=grafana -p 3000:3000 grafana/grafana
```

A data source must be set to Cortex in order for the data to show up in Grafana. Go to the
data sources section in Grafana and select the Prometheus option. Add
`http://host.docker.internal:9009/api/prom` as the URL and clock save and exit. A green
box should pop up stating that the data source is valid. The username / password is
admin/admin. There may be a password reset screen which can be ignored. See the example
project README for more information on how to setup and use Grafana.

### Step 3 - Run Pipeline Test
The last step is to run the pipeline test in `pipeline.go`. Modify `main.go` to select
which tests to run and then run the tests with:

`go build` followed by `./pipeline`
