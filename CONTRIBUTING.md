# Contributing

## Setup

```bash
corepack enable
pnpm install
cp .env.example .env
```

Fill in `.env` before running any app — see [README.md](README.md) for details.

## Workflow

1. Create a branch off `main`.
2. Make your changes, following the conventions in [CLAUDE.md](CLAUDE.md) (Gin layout
   in the Go services, Feature-Sliced Design on the frontends, absolute imports in
   TypeScript, VeeValidate on forms).
3. Before opening a PR, run:
   ```bash
   pnpm lint
   pnpm typecheck
   pnpm build
   pnpm test:go
   ```
4. If you changed a service's database schema, add a goose SQL migration to its
   `internal/migrations` (applied automatically on start). If you changed a `.proto`,
   run `pnpm proto:generate` and commit the generated code.
5. Open a PR against `main` using the PR template checklist.

## Commit messages

Use a short imperative summary, optionally prefixed with a type (`fix:`, `feat:`,
`chore:`, `refactor:`), e.g. `fix: correct admin JWT expiry`.
