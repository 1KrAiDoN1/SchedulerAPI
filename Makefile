DOCKER_COMPOSE = docker-compose

# Docker команды
docker-build: ## Собрать Docker образы
	$(DOCKER_COMPOSE) build

docker-up: ## Запустить все сервисы
	$(DOCKER_COMPOSE) up -d

docker-down: ## Остановить все сервисы
	$(DOCKER_COMPOSE) down

docker-logs: ## Показать логи всех сервисов
	$(DOCKER_COMPOSE) logs -f

docker-logs-scheduler: ## Показать логи планировщика
	$(DOCKER_COMPOSE) logs -f scheduler

docker-logs-worker: ## Показать логи воркера
	$(DOCKER_COMPOSE) logs -f worker

docker-logs-db: ## Показать логи базы данных
	$(DOCKER_COMPOSE) logs -f db

docker-ps: ## Показать статус контейнеров
	$(DOCKER_COMPOSE) ps

docker-restart: ## Перезапустить все сервисы
	$(DOCKER_COMPOSE) restart

docker-stop: ## Остановить все сервисы
	$(DOCKER_COMPOSE) stop

docker-start: ## Запустить остановленные сервисы
	$(DOCKER_COMPOSE) start


