include .env
export

migrate-up:
	migrate -path migrations -database "$(DB_URL)?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)?sslmode=disable" down

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

.PHONY: migrate-up migrate-down migrate-create