# FrameCheck backend (planned)

Go API with Clean Architecture and PostgreSQL migrations:

```text
internal/
  domain/             Pure entities and validation
  application/        Use cases
  ports/              Persistence, storage, inference interfaces
  adapters/http/      HTTP transport
  adapters/postgres/  Database repository
  adapters/storage/   Local asset store
  adapters/meta/      Hosted inference adapter
migrations/           Versioned PostgreSQL schema changes
```

Dependencies point inward. Model responses and image coordinates are validated
before persistence. See [architecture](../docs/ARCHITECTURE.md) and
[contracts](../docs/CONTRACTS.md). Implemented with an offline mock provider; the Meta adapter is not written yet. Run `go run ./cmd/api` (port 8089) and `go test ./...`.
