DATABASE_NAME:=gophermart

run:
	go run cmd/gophermart/main.go 

t:
	go test ./...

mocks:
	mockgen -source=internal/pkg/repository/repository.go Repository > internal/pkg/test/mocks/repository_mock.go

db-create:
	psql -U postgres -c "drop database if exists $(DATABASE_NAME)"
	psql -U postgres -c "create database $(DATABASE_NAME)"

db-up:
	goose -dir ./internal/pkg/database/migrations postgres "${DATABASE_URI}" up
