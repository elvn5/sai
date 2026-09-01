# CLAUDE.md

## General Rules & Code Style
- **Imports:** ALWAYS use absolute imports (`@/` or `~/`) instead of relative paths (`../`, `./`).
- **Architecture:**
  - **Backend:** Clean Architecture (Entities -> Use Cases -> Controllers / Interface Adapters -> Frameworks & Drivers). Keep business logic strictly independent of frameworks/DBs.
  - **Frontend:** Feature-Sliced Design (FSD) (`app/`, `pages/`, `widgets/`, `features/`, `entities/`, `shared/`). Respect strict cross-layer import rules.
- **Formatting & Linting:** Use **Biome** for formatting and linting (`npx @biomejs/biome check --write`).
- **Validation:** Use **Zod** (backend) and **VeeValidate** (frontend) for input/data validation.
- **Logging:** Use **Pino** for all logging. Avoid `console.log`.
- **Environment:** Manage environment variables via `dotenv` (`.env`).
- **TypeScript:** Strict mode enabled. Always specify return types for public functions/methods.

## Tech Stack
- **Backend:** Node.js v24, Next.js / Express, TypeORM, PostgreSQL, Axios, Socket.io, CORS, Docker
- **Frontend:** Nuxt 3, Pinia (+ persistedstate), @nuxt/ui, Tailwind CSS, Nuxt Icon, @nuxtjs/seo, @nuxtjs/i18n, @vueuse/nuxt, @tanstack/vue-query
- **Linter/Formatter:** Biome

## Commands
- **Dev:** `npm run dev` / `pnpm dev`
- **Build:** `npm run build` / `pnpm build`
- **Lint/Format:** `npx @biomejs/biome check --write .`
- **Type Check:** `npx tsc --noEmit`
- **Docker:** `docker compose up -d --build`
