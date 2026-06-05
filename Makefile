.PHONY: up down build run seed migrate3 migrate4 migrate5 migrate6 tidy lint

up:
	docker compose up -d

down:
	docker compose down

build:
	docker compose build

run:
	go run ./cmd/server/main.go

seed:
	docker exec -i workplace_postgres psql -U postgres -d workplace < migrations/002_seed.sql

migrate3:
	docker exec -i workplace_postgres psql -U postgres -d workplace < migrations/003_add_user_role.sql

migrate4:
	docker exec -i workplace_postgres psql -U postgres -d workplace < migrations/004_user_position_dept_name.sql

migrate5:
	docker exec -i workplace_postgres psql -U postgres -d workplace < migrations/005_new_features.sql

migrate6:
	docker exec -i workplace_postgres psql -U postgres -d workplace < migrations/006_remove_user_role.sql

tidy:
	go mod tidy

lint:
	golangci-lint run ./...

test:
	go test ./... -v
