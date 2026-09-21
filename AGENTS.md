<!--VITE PLUS START-->

# Using Vite+, the Unified Toolchain for the Web

This project is using Vite+, a unified toolchain built on top of Vite, Rolldown, Vitest, tsdown, Oxlint, Oxfmt, and Vite Task. Vite+ wraps runtime management, package management, and frontend tooling in a single global CLI called `vp`. Vite+ is distinct from Vite, and it invokes Vite through `vp dev` and `vp build`. Run `vp help` to print a list of commands and `vp <command> --help` for information about a specific command.

Docs are local at `node_modules/vite-plus/docs` or online at https://viteplus.dev/guide/.

## Review Checklist

- [ ] Run `bun install --frozen-lockfile` from the repository root before getting started.
- [ ] Run `bun run check`, `bun run test`, and `bun run build` from the root. Turbo runs Vite+ in `frontend/` and native Go tasks in `backend/`.
- [ ] Check if there are `vite.config.ts` tasks or `package.json` scripts necessary for validation, run via `vp run <script>`.

<!--VITE PLUS END-->

## Workspace orchestration

Turborepo 2.11 uses experimental native Go workspace discovery (`go.work`) and task command overrides. Keep the version pinned. The frontend uses Bun and Vite+; the backend needs no `package.json`.

Run `bun install --frozen-lockfile` and `bun run <task>` from the repository root. Use `--filter=frontend` or `--filter=./backend` to select an app. Go must be installed even for filtered frontend Turbo commands, since Turbo discovers the whole workspace. Direct `vp` commands must still run inside `frontend/`.

## Running the application

Browser verification requires both the backend and frontend.

1. Configure `backend/.env` from `.env.sample` and the frontend environment as needed.
2. Start Postgres with `make -C backend db-up` and apply migrations with `make -C backend migrate-up`.
3. Run `bun run dev` from the root to start both applications. Backend dev uses `make run` to load `backend/.env`.

Database lifecycle and migration commands remain explicit Make targets; Turbo does not run them automatically.

### Before browser testing

- Only do browser testing if explicitly asked by the user
- Reuse services that are already running.
- Confirm that both the backend and frontend have started successfully.
- Use the development-only `dev-ui` route instead of Google login.
- If `dev-ui` is missing, add it for testing.
- Check browser console errors and failed network requests.
