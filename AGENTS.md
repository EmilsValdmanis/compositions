<!--VITE PLUS START-->

# Using Vite+, the Unified Toolchain for the Web

This project is using Vite+, a unified toolchain built on top of Vite, Rolldown, Vitest, tsdown, Oxlint, Oxfmt, and Vite Task. Vite+ wraps runtime management, package management, and frontend tooling in a single global CLI called `vp`. Vite+ is distinct from Vite, and it invokes Vite through `vp dev` and `vp build`. Run `vp help` to print a list of commands and `vp <command> --help` for information about a specific command.

Docs are local at `node_modules/vite-plus/docs` or online at https://viteplus.dev/guide/.

## Review Checklist

- [ ] Run `vp install` after pulling remote changes and before getting started.
- [ ] Run `vp check` and `vp test` to format, lint, type check and test changes.
- [ ] Check if there are `vite.config.ts` tasks or `package.json` scripts necessary for validation, run via `vp run <script>`.

<!--VITE PLUS END-->

## Running the application

Browser verification requires both the backend and frontend.

All commands below must be run from the `/backend` directory, **not the repository root**.

### 1. Start the database

```bash
cd backend
make db-up
```

### 2. Start both applications

Open two separate terminals. In **both terminals**, change to the `/backend` directory first.

Backend:

```bash
cd backend
make run
```

Frontend:

```bash
cd frontend
vp dev
```

Do not run `vp dev` or other `vp` commands from the repository root.

### Before browser testing

- Reuse services that are already running.
- Confirm that both the backend and frontend have started successfully.
- Use the development-only `dev-ui` route instead of Google login.
- If `dev-ui` is missing, add it for testing.
- Check browser console errors and failed network requests.
