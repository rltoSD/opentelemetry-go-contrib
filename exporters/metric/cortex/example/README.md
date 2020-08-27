# Cortex Exporter Example

<<<<<<< HEAD
This example exports several metrics to a Cortex instance.

## Instructions

### Requirements

- [Docker Compose](https://docs.docker.com/compose/) installed

1. Run the docker container
=======
This example exports several metrics to a [Cortex](https://cortexmetrics.io/) instance and displays
them in [Grafana](https://grafana.com/).

## Requirements

- [Docker Compose](https://docs.docker.com/compose/) installed

## Instructions

1. Run the docker container with the following command
>>>>>>> upstream-master

```bash
docker-compose up -d
```

<<<<<<< HEAD
2. Log into the Grafana instance running at `http://localhost:3000`. The login credentials are
   admin/admin.

3. Add Cortex as a data source by creating a new Prometheus data source is using
   `http://cortex:9009/api/prom/` as the endpoint. Because Cortex is running in a docker container,
   we use `cortex` as the url instead of `localhost`.
=======
2. Log in to the Grafana instance running at [http://localhost:3000](http://localhost:3000). The
   login credentials are admin/admin.

3. Add Cortex as a data source by creating a new Prometheus data source using
   [http://localhost:9009/api/prom/](http://localhost:9009/api/prom/) as the endpoint.
>>>>>>> upstream-master

4. View collected metrics in Grafana.

5. Shut down the services when you're finished with the example

```bash
docker-compose down
```
