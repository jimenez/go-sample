all:
	docker compose up --build -d

TESTFILES = $(wildcard 	./integration_tests/*.sh)

test:
	@echo "Running unit tests"
	@go test ./...

	@echo "Running integration tests"
	@docker compose stop > /dev/null 2>&1
	@docker compose rm -f > /dev/null 2>&1
	@docker volume rm -f go-sample_dbvolume > /dev/null 2>&1
	@docker compose up --build -d > /dev/null 2>&1
	@for FILE in $(TESTFILES); do $$FILE; done

	@docker compose stop > /dev/null 2>&1