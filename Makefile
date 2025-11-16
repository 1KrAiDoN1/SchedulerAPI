DOCKER_COMPOSE = docker-compose

# Переменные окружения для подключения к БД
include .env
export

DB_HOST = localhost
DB_PORT = 5435
MIGRATIONS_DIR = migrations

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

docker-logs-nats: ## Показать логи NATS
	$(DOCKER_COMPOSE) logs -f nats

docker-ps: ## Показать статус контейнеров
	$(DOCKER_COMPOSE) ps

docker-restart: ## Перезапустить все сервисы
	$(DOCKER_COMPOSE) restart

docker-stop: ## Остановить все сервисы
	$(DOCKER_COMPOSE) stop

docker-start: ## Запустить остановленные сервисы
	$(DOCKER_COMPOSE) start

# Команды миграций
migrate-up: ## Применить все миграции к базе данных
	@echo "Применение миграций к базе данных..."
	@for migration in $$(ls $(MIGRATIONS_DIR)/*.up.sql | sort); do \
		echo "Применение: $$(basename $$migration)"; \
		PGPASSWORD=$(POSTGRES_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(POSTGRES_USER) -d $(POSTGRES_DB) -f $$migration || exit 1; \
		echo "✓ Успешно применена: $$(basename $$migration)"; \
	done
	@echo "✓ Все миграции успешно применены!"

migrate-down: ## Откатить все миграции (удалить все таблицы)
	@echo "Откат всех миграций..."
	@for migration in $$(ls $(MIGRATIONS_DIR)/*.down.sql | sort -r); do \
		echo "Откат: $$(basename $$migration)"; \
		PGPASSWORD=$(POSTGRES_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(POSTGRES_USER) -d $(POSTGRES_DB) -f $$migration || exit 1; \
		echo "✓ Успешно откачена: $$(basename $$migration)"; \
	done
	@echo "✓ Все миграции успешно откачены!"

migrate-status: ## Проверить статус миграций (показать существующие таблицы)
	@echo "Текущие таблицы в базе данных:"
	@PGPASSWORD=$(POSTGRES_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(POSTGRES_USER) -d $(POSTGRES_DB) -c "\dt"



