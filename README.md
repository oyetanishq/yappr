<h1 align="center">Yappr</h1>

<p align="center">AI-powered code review for GitHub pull requests.</p>

Yappr is an AI-powered code review tool designed to provide insightful, automated feedback on your codebase. Install the Yappr GitHub App on a repository and every pull request gets an automated, LLM-generated review - a PR summary, per-file analysis, an architecture/flow diagram, and evidence-based bug detection - posted straight back as a PR comment.

## Features

- **Automated PR reviews** - on every `pull_request` event, an agent clones the repo, diffs the PR, and runs a multi-pass LLM review pipeline.
- **Three-pass review pipeline:**
  - PR summary + per-file analysis
  - Mermaid flow diagram of the new execution path
  - Deep, evidence-based bug detection using a stronger model
- **PR review history** - every reviewed PR is persisted as a run and surfaced in the dashboard. Each run tracks its lifecycle status (`processing` → `completed` / `failed` / `limit_reached`), diff stats (files changed, additions, deletions), and the full output of all three LLM passes.
- **In-dashboard review viewer** - browse past runs, expand any one to read the summary, per-file analysis, and bug report, and view the **Mermaid flow diagram rendered client-side** (not just as raw text) — with a direct link back to the PR on GitHub.
- **Customizable personalities** - tailor the reviewer's tone to your team's culture: `bestie`, `senior_dev`, `sigma`, `toxic_tech_lead`.
- **Per-repo configuration** - ignore paths (e.g. `dist/`, `node_modules/`) and choose a personality per repository.
- **GitHub OAuth + App install** - sign in with GitHub, then install the App and pick which repos it can access.
- **Session management** - JWT session cookies backed by Redis, with multi-device session listing and revocation.
- **Free / Pro tiers** - billing via Razorpay (subscribe / cancel / resume) with webhook-driven plan activation. Some personalities and features are Pro-only.

## Tech stack

| Layer | Technology |
|---|---|
| **Backend** | Go 1.26, [Gin](https://github.com/gin-gonic/gin), structured logging via [zap](https://github.com/uber-go/zap) |
| **Datastores** | MongoDB 7 (source of truth), Redis 7 (sessions + caches) |
| **Frontend** | React 19, Vite, Tailwind CSS, Zustand, React Router, lucide-react |
| **LLM** | OpenAI-compatible API ([go-openai](https://github.com/sashabaranov/go-openai)); default provider is Google Gemini (`gemini-2.5-flash` / `gemini-2.5-pro`), configurable via `LLM_BASE_URL` |
| **Auth** | GitHub App + OAuth, JWT (HS256) session cookies |
| **Billing** | Razorpay subscriptions (HMAC-verified webhooks) |
| **Load testing** | [k6](https://k6.io/) with TypeScript scenarios |
| **Infra** | Docker Compose, Go workspaces (`go.work`) |

## Repository structure

A monorepo of independently-built apps under `apps/`, wired together with a Go workspace (`go.work`) and Docker Compose.

```text
yappr/
├── apps/
│   ├── api/        # REST API (Go/Gin)         — :8080
│   ├── agent/      # PR review worker (Go/Gin)  — :8081
│   ├── web/        # Frontend (React + Vite)    — :5173
│   ├── shared/     # Shared Go module (config, db, models, helpers)
│   ├── scripts/    # One-off Go tools (DB seeder for load tests)
│   └── k6/         # k6 stress-test suite (TypeScript)
├── docker-compose.yml
├── go.work
└── .github/workflows/stress-test.yml
```

| App | What it does |
|---|---|
| **`apps/api`** | Public REST API: GitHub OAuth login, session/auth, GitHub App installations, per-repo config, PR review run history (`GET /api/v1/runs`, `GET /api/v1/runs/:id`), and Razorpay billing. Owns the user-facing surface. |
| **`apps/agent`** | Receives GitHub App webhooks (`POST /api/v1/github/webhook`), runs the review pipeline (clone → diff → 3 LLM passes → post comment), and records each run's progress and result as a `PRRun` document. HMAC-verifies webhook signatures. |
| **`apps/web`** | Dashboard SPA — sign in, install the App, pick repos, configure reviews, browse PR review history with client-rendered Mermaid diagrams, and manage billing. |
| **`apps/shared`** | Shared Go code imported by api/agent/scripts: config loader, Mongo/Redis clients, data models, and the JSON response envelope. |
| **`apps/scripts`** | The `seed` tool — bulk-creates users, sessions, and repo configs in Mongo + Redis and emits `users.json` for the load tests. |
| **`apps/k6`** | Open-model stress tests covering health, auth, repo config, GitHub installations, and billing routes (the slow LLM review path is intentionally excluded). |

## Getting Started

### Prerequisites

- A **GitHub App** (App ID, client ID/secret, private key, webhook secret, app name).
- An **LLM API key** (Google Gemini by default, or any OpenAI-compatible endpoint).
- A **Razorpay** account (key ID/secret, a monthly plan ID, webhook secret) — only needed for billing flows.
- Docker + Docker Compose **or** Go 1.26, Node 22 + pnpm, and local Redis/Mongo.

### Environment files

```bash
cp apps/api/.env.example   apps/api/.env
cp apps/agent/.env.example apps/agent/.env
```

Fill in the GitHub App, LLM, and Razorpay values. Key variables:

- **API** (`apps/api/.env`): `PORT=8080`, `JWT_SECRET`, `REDIS_URL`, `MONGODB_URI`, `MONGODB_DB`, GitHub App creds, Razorpay creds.
- **Agent** (`apps/agent/.env`): `PORT=8081`, `REDIS_URL`, `MONGODB_URI`, GitHub App creds, `LLM_API_KEY`, `LLM_BASE_URL`, `LLM_BASE_MODEL`, `LLM_BUG_MODEL`.

> The GitHub private key is stored base64-encoded. On macOS: `base64 -i ./keys/github-key.pem` and paste into `GITHUB_APP_PRIVATE_KEY`. `JWT_SECRET` **must match** across the API and the seeder, or seeded session cookies won't validate.

### Run with Docker (recommended)

```bash
# Build & start the full stack (Redis, Mongo, API, Agent, Web)
docker compose up -d --build
```

| Service | URL |
|---|---|
| Web (dashboard) | <http://localhost:5173> |
| API | <http://localhost:8080> |
| Agent | <http://localhost:8081> |
| RedisInsight | <http://localhost:5540> |

Tear down (this **wipes** Redis/Mongo data — you'll need to re-seed before load testing):

```bash
docker compose down -v
```

### Run locally (without Docker)

Start Redis and Mongo yourself (e.g. `docker compose up -d redis mongo`), then in separate terminals:

```bash
# API  → :8080
go run ./apps/api/cmd/server

# Agent → :8081
go run ./apps/agent/cmd/agent

# Web  → :5173
cd apps/web && pnpm install && pnpm dev
```

> For webhooks to reach the agent from GitHub during local dev, expose it with a tunnel (e.g. `ngrok` / `cloudflared`) and point the GitHub App's webhook URL at it.

## Stress testing

Load tests live in `apps/k6` (TypeScript, bundled with esbuild). They use an **open-model** `ramping-arrival-rate` executor to push request rate up until the system saturates, and cover every route **except** the LLM review path (which is bound by external model latency).

### Run the suite

```bash
# 1) Start the stack + seed data (REQUIRED — an unseeded DB makes every
#    authenticated request 401, invalidating the whole run)
docker compose up -d --build --wait redis mongo api agent
docker compose --profile seed run --build --rm seed

# 2) Run k6 (tune load via -e overrides)
docker compose --profile test run --build -T --rm \
  -e TARGET_RPS=400 -e PEAK_RPS=1500 k6
```

Tunable env vars: `TARGET_RPS`, `PEAK_RPS`, `RAMP`, `HOLD`, `MAX_VUS`, `PRE_VUS`. Each iteration fires ~12–15 HTTP requests, so real req/s ≈ `rate × ~13`.

### Test machine

| | |
|---|---|
| **Machine** | MacBook, Apple **M3** (8-core) |
| **Memory** | 16 GB |
| **Storage** | 512 GB SSD |
| **Setup** | Full stack + k6 load generator co-located in Docker Desktop |

> All containers, both databases, **and** the k6 generator share the same 8 cores, so these numbers reflect a single laptop — not a production ceiling. For true limits, run k6 from a separate machine against a deployed API.

### Latest results

`~7,800 req/s` sustained, `p95 = 21 ms`, `p99 = 39 ms`, **100% of checks passing**:

```text
checks_total ....: 478,941   (100.00% succeeded)
http_reqs .......: 468,941   7,764 req/s
http_req_duration: p(95)=21.1ms  p(99)=38.52ms
iterations ......: 22,747    377 iters/s
```

> **Reading `http_req_failed`:** the suite reports ~72% "failed" requests, but that is **expected and healthy**. Most checks deliberately assert error responses — `401` (missing/invalid auth), `402` (Pro-only feature on a Free account), `404`, `400`, `413` (oversized webhook body). k6 counts every non-2xx as "failed" even when it's the intended outcome. The real signal is **`checks` at 100%** — every endpoint returned exactly the status it should.

The suite runs two scenarios concurrently:

- **`stress`** — open-model, idempotent read/verify workload driven at high RPS.
- **`destructiveAuth`** — the logout / session-revoke flow, run once per seeded user in a bounded scenario (kept out of the high-RPS loop so consumed sessions don't produce false failures).

CI runs a lightweight version of this suite on every push/PR via [`.github/workflows/stress-test.yml`](.github/workflows/stress-test.yml).
