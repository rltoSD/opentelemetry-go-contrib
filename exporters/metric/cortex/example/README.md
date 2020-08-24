# Cortex Exporter Example

This example exports several metrics to a Cortex instance.

## Instructions

### Requirements

- [Docker Compose](https://docs.docker.com/compose/) installed

1. Run the docker container

```bash
docker-compose up -d
```

2. Log into the Grafana instance running at `http://localhost:3000`. The login credentials are
   admin/admin.

3. Add Cortex as a data source by creating a new Prometheus data source is using
   `http://cortex:9009/api/prom/` as the endpoint. Because Cortex is running in a docker container,
   we use `cortex` as the url instead of `localhost`.

4. View collected metrics in Grafana.

5. Shut down the services when you're finished with the example

```bash
docker-compose down
```
