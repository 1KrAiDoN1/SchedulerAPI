DOCKER_COMPOSE = docker-compose

docker-build:
	$(DOCKER_COMPOSE) build

docker-up: 
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-logs:
	$(DOCKER_COMPOSE) logs -f

docker-ps:
	$(DOCKER_COMPOSE) ps

	