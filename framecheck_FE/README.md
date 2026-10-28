# FrameCheck frontend (planned)

React, TypeScript, Vite, and Tailwind CSS. Use Clean Architecture:

```text
src/
  domain/             Pure entities and rules
  application/        Use cases and port interfaces
  adapters/api/       HTTP adapter and DTO mapping
  adapters/ui/        React pages, components, view models, styling
  app/                Routing and dependency composition
```

Dependencies point inward. React components call use cases and do not own
network calls or domain validation. See [architecture](../docs/ARCHITECTURE.md).
Implemented: `pnpm dev` (port 5181, proxies `/v1` to the API), `pnpm test`, `pnpm build`.
