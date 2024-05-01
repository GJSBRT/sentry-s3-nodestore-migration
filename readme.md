# Sentry S3 Nodestore migration util
This is a veeery simple script which will migrate your nodestore data from postgres to your s3 provider. This can also be used if you'd just like to export your nodestore data to s3.
This script is meant to be used for people migrating their Sentry to s3 using https://github.com/kanadaj/sentry-s3-nodestore 

## Speed
Mirgating events can take a long time if you have many events. I have migrated about 22 milion events over several days. The speed of migration is highly depended on the speed your S3 provider.

If migrating an event took 50ms. If we multiply that by 22 milion events then we get ~12 days of continous migrating. So this is something to keep in mind.

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
  -db string
        postgres db url. e.g. postgres://postgres:@localhost:5432 (default "postgres://postgres:@localhost:5432")
  -debug
        debug mode shows more infomation
  -limit int
        max amount of rows to parse at once (default 1000)
  -offset int
        offset to start at
  -s3bucket string
        s3 bucket name
  -s3domain string
        s3 provider domain eg. s3.example.com
  -s3key string
        s3 access key.
  -s3secret string
        s3 secret
```

## Useful commands
Go in to postgres
```sh
docker exec -it sentry-self-hosted-postgres-1 psql -U postgres
```
Count rows
```sql
select count(*) from public.nodestore_node;
```
Get size of nodestore
```sql
select pg_size_pretty(pg_total_relation_size('public.nodestore_node'));
```
Get size of largest event in kb
```sql
select id, pg_column_size(data) / 1000 as row_size from public.nodestore_node order by row_size desc limit 1;
```
