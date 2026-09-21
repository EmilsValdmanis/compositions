<p align="center">
  <img src="frontend/public/favicon.svg" alt="Compositions joker logo" width="96" height="96">
</p>

<h1 align="center">Compositions</h1>

<p align="center">
  A family card game brought online, with real-time rooms, server-backed game state, and careful joker handling.
</p>

<p align="center">
  <a href="RULES.md">Game Rules</a>
  ·
  <a href="frontend/README.md">Frontend</a>
  ·
  <a href="backend/README.md">Backend</a>
</p>

## The Game

Compositions is a multi-round card game built around:

- forming valid sets and runs
- hitting the initial 40-point requirement
- managing jokers and high/low aces carefully
- getting rid of your hand before everyone else

The full rule set lives in [RULES.md](RULES.md).

## Repository Layout

- `frontend/`: the web app built with React, TanStack Start, and Vite+
- `backend/`: the Go server and game logic module, including Postgres persistence via `pgx` + `sqlc`
- `RULES.md`: the current written rules for the game

## Current Status

The base game is implemented and playable. Current focus areas:

- improving the pre-game and post-game experience
- refining in-game interactions, animations, and sound
- adding social/statistics features now that persistence is in place

## Working In This Repo

Install Bun 1.4.2, Go 1.26.6 or newer, and Node.js 24 (for frontend tooling). Run from the repository root:

```bash
bun install --frozen-lockfile
make -C backend db-up
make -C backend migrate-up
bun run dev
```

Configure `backend/.env` from `backend/.env.sample` first. The frontend listens on port 3000 and the backend on 8080. Docker is required for local Postgres and integration tests.

```bash
bun run check
bun run test
bun run build
bun run test-integration
# Run only one application:
bun run test --filter=frontend
bun run build --filter=./backend
```

Turborepo 2.11.0 orchestrates the Bun frontend workspace and the native Go module in `go.work`. Its Go workspace and command override features are experimental, so Turbo is pinned. Go must be available even for filtered frontend Turbo commands. The Go package name is `compositions`, derived from its module path; directory filters avoid confusion with the repository name.

Vite+ remains the frontend builder, formatter, linter, type checker, and test runner. Use direct `vp` commands inside `frontend/` only. Bun dependencies and overrides are managed by the root `package.json` and `bun.lock`.

Turbo caches backend binaries (`backend/dist/server`), checks, unit tests, and generated translations. Frontend builds deliberately run every time because the Sentry plugin can upload sourcemaps. Development, formatting, and integration tests are uncached. Backend build hashes include the Go toolchain and target; frontend build/test configuration tracks its `.env` files and build variables.

Database lifecycle, migrations, benchmarks, and SQL generation remain explicit `make -C backend <target>` commands. Production Railway deployment still uses its standalone Go build and migration commands. For Vercel, keep `frontend` as the project root and enable access to files outside that directory so installation can use the root Bun workspace and lockfile.


## Remote Task Cache

GitHub Actions uses [Vercel Remote Cache](https://github.com/vercel/setup-turborepo-remote-cache-action) for both frontend and backend Turbo tasks. The action exchanges GitHub's OIDC identity for a short-lived cache token and revokes it when the job finishes. No long-lived `TURBO_TOKEN` repository secret is needed.

One-time account setup:

1. In your Vercel team's **Settings → Build and Deployment → OIDC Policies for CLI Access**, add a **Turborepo CLI Policy** for GitHub repository `EmilsValdmanis/compositions`. Restrict it to workflow `.github/workflows/pr-validation.yml`. Allow the PR refs you intend to validate, as well as any branches used for manual workflow runs.
2. Set the GitHub Actions repository **variable** `TURBO_TEAM` to that team's slug or ID:

   ```bash
   gh variable set TURBO_TEAM --repo EmilsValdmanis/compositions --body YOUR_TEAM_SLUG
   ```

The cache setup runs for same-repository PRs and manual workflow runs. Fork PRs, Dependabot runs, and repositories without `TURBO_TEAM` continue with local task caching. Once `TURBO_TEAM` is configured, authentication failures fail the job so a broken connection is visible.

For local access, run from the repository root and select the same team:

```bash
bunx turbo login
bunx turbo link
```

The local link is stored in gitignored `.turbo/config.json`; credentials are stored outside the repository. To verify remote writes and reads independently of your local cache, run this twice:

```bash
bunx turbo run check --cache=remote:rw
```

Look for `Remote caching enabled` and cache hits on the second run. A first-run hit is also valid if another machine already populated the cache. Go cache reuse depends on matching toolchain and target settings, so different operating systems or Go versions can produce misses.

Remote caching uses the existing task policies: checks, unit tests, translation generation, and backend builds are cached. Frontend builds, integration tests, development servers, and formatting remain uncached. Go coverage enforcement, race detection, static analysis, vulnerability scanning, and fuzzing in CI remain separate commands outside Turbo's cache.
