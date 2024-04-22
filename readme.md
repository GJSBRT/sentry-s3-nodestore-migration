# Sentry S3 Nodestore migration util
This is a veeery simple script which will migrate your nodestore data from postgres to your s3 provider. This can also be used if you'd just like to export your nodestore data to s3.
This script is meant to be used for people migrating their Sentry to s3 using https://github.com/kanadaj/sentry-s3-nodestore 

## Running it
There are a few things you will need to do.

1. Binding postgres to a port. Add the following lines to your `docker-compose.yaml` file and rerun the `./install.sh` afterwards.
```yaml
  postgres:
    <<: *restart_policy
    # Using the same postgres version as Sentry dev for consistency purposes
    image: "postgres:14.11-alpine"
    ports:              # <-- Add these lines
      - "5432:5432/tcp" # <-- Add these lines
```

2. Startup the postgres container and nothing else. 
```sh
docker compose up -d postgres
```

3. Run this tool using these parameters.
```sh
sentry-s3-nodestore-migration \
    --db postgres://postgres:@localhost:5432 \ # (optional)
    --s3domain s3.example.com \
    --s3key YOURKEY \
    --s3secret YOURSECRET \
    --s3bucket BUCKETNAME
```

## Useful commands
Go in to postgres
```sh
docker exec -it sentry-self-hosted-postgres-1 psql -U postgres
```
Count rows
```sh
postgres=# select count(*) from public.nodestore_node;
```
