# devexperience

Setup the storage by running

``docker-compose up``

```
docker exec -it workshop-cs cqlsh

CREATE KEYSPACE stats WITH REPLICATION = {'class' : 'SimpleStrategy', 'replication_factor' : 1 };
DESCRIBE KEYSPACES;
USE stats;
CREATE TABLE metrics (
   metric_id varchar,
   ts timestamp,
   service_id varchar,
   min double,
   max double,
   avg double,
   PRIMARY KEY (metric_id, ts, service_id)
);
CREATE TABLE rollups300 (
   metric_id varchar,
   ts timestamp,
   service_id varchar,
   min double,
   max double,
   avg double,
   PRIMARY KEY (metric_id, ts, service_id)
);
DESCRIBE TABLES;
exit
```
