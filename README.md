* Run `docker-compose up`, if you're using `rabbitmq` or `kafka` you may see the _rest-server_ and _elasticsearch-indexer_ services fail because those services take too long to start, in that case use any of the following instructions to manually start those services after the dependent server is ready:
    * If you're planning to use RabbitMQ, run `docker-compose up rest-server elasticsearch-indexer-rabbitmq`.
    * If you're planning to use Kafka, run `docker-compose up rest-server elasticsearch-indexer-kafka`.
* For building the service images you can use:
    * `rest-server` image: `docker-compose build rest-server`.
    * `elasticsearch-indexer-rabbitmq` image: `docker-compose build elasticsearch-indexer-rabbitmq`.
    * `elasticsearch-indexer-kafka` image: `docker-compose build elasticsearch-indexer-kafka`.
    * `elasticsearch-indexer-redis` image: `docker-compose build elasticsearch-indexer-redis`.
* Run `docker-compose run rest-server tern migrate --migrations "/api/migrations/" --conn-string "postgres://user:password@postgres:5432/dbname?sslmode=disable"` to have everything working correctly.
* Finally interact with the API using Swagger UI: http://127.0.0.1:9234/static/swagger-ui/
