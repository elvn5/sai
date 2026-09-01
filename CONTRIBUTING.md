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
2. Make your changes, following the conventions in [CLAUDE.md](CLAUDE.md) (absolute
   imports, Clean Architecture on the backend, Feature-Sliced Design on the admin
   frontend, Zod/VeeValidate for validation, Pino for logging).
3. Before opening a PR, run:
   ```bash
   pnpm lint
   pnpm typecheck
   pnpm build
   ```
4. If you changed the database schema, generate a migration:
   ```bash
   pnpm migration:generate
   ```
5. Open a PR against `main` using the PR template checklist.

## Commit messages

Use a short imperative summary, optionally prefixed with a type (`fix:`, `feat:`,
`chore:`, `refactor:`), e.g. `fix: correct admin JWT expiry`.
