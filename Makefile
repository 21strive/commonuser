.PHONY: migrate migrate-help migrate-build clean

# Build the migration tool
migrate-build:
	@mkdir -p migrate/bin
	@echo "Building migration tool..."
	@rm -f migrate/bin/migrate
	cd migrate && CGO_ENABLED=0 go build -v -o bin/migrate migrate.go
	@chmod +x migrate/bin/migrate
	@echo "Build successful"
	@file migrate/bin/migrate

# Run migration with default parameters (requires DB credentials)
migrate: migrate-build
	@echo "Usage: make migrate ENTITY=<entity> USER=<user> PASSWORD=<password> DB=<database> [HOST=<host>] [PORT=<port>] [SSL=<ssl>]"
	@echo "Example: make migrate ENTITY=user USER=postgres PASSWORD=mypass DB=myapp"
	@echo ""
	@if [ -z "$(ENTITY)" ] || [ -z "$(USER)" ] || [ -z "$(DB)" ]; then \
		echo "Error: Missing required parameters. Use 'make migrate-help' for usage."; \
		exit 1; \
	fi
	@echo "Running migration..."
	./migrate/bin/migrate -entity=$(ENTITY) -user=$(USER) -db=$(DB) -password=$(PASSWORD) -host=$(HOST) -port=$(PORT) -ssl=$(SSL) || true
	rm -rf migrate/bin

# Show detailed help
migrate-help:
	@echo "Migration Tool Usage:"
	@echo ""
	@echo "Required parameters:"
	@echo "  ENTITY    - Entity name (e.g., 'user', 'admin')"
	@echo "  USER      - Database user"
	@echo "  DB        - Database name"
	@echo ""
	@echo "Optional parameters:"
	@echo "  PASSWORD  - Database password"
	@echo "  HOST      - Database host (default: localhost)"
	@echo "  PORT      - Database port (default: 5432)"
	@echo "  SSL       - SSL mode: disable, require (default: disable)"
	@echo ""
	@echo "Examples:"
	@echo "  make migrate ENTITY=user USER=postgres PASSWORD=secret DB=prod HOST=db.example.com SSL=require"

# Clean built binaries
clean:
	rm -rf migrate/bin