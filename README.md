# Workplace App — Backend API

A Go-based REST API for managing workplace operations: attendance, projects, memos, HR workflows, and real-time communication.

---

## Table of Contents

- [Overview](#overview)
- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
- [Getting Started](#getting-started)
- [Environment Variables](#environment-variables)
- [Running the Project](#running-the-project)
- [API Overview](#api-overview)
- [Modules](#modules)
- [Database Migrations](#database-migrations)
- [Background Jobs](#background-jobs)
- [Authentication](#authentication)
- [WebSockets](#websockets)
- [Geofencing & Device Lock](#geofencing--device-lock)
- [Makefile Commands](#makefile-commands)
- [Default Credentials](#default-credentials)

---

## Overview

Workplace App is an internal company management platform. It provides:

- Role-based access control (super admin, department head, staff, procurement)
- Employee attendance tracking with GPS geofencing
- Project and task management with milestones and budgets
- Internal memos with per-user acknowledgement tracking
- Real-time chat (direct messages and group rooms)
- Real-time in-app notifications with email fallback
- HR workflows: leave requests, performance reviews, expense claims, asset tracking
- Automated daily background jobs for reminders and attendance auto-close
- Analytics and reporting

---

## Tech Stack

| Layer            | Technology                                                     |
| ---------------- | -------------------------------------------------------------- |
| Language         | Go 1.22                                                        |
| HTTP Router      | [chi v5](https://github.com/go-chi/chi)                        |
| Database         | PostgreSQL 16 via [pgx v5](https://github.com/jackc/pgx)       |
| Cache            | Redis 7 via [go-redis v9](https://github.com/redis/go-redis)   |
| WebSockets       | [gorilla/websocket](https://github.com/gorilla/websocket)      |
| Auth             | JWT via [golang-jwt/jwt v5](https://github.com/golang-jwt/jwt) |
| Email            | SMTP (MailHog in development)                                  |
| Containerisation | Docker + Docker Compose                                        |
| Reverse Proxy    | Nginx (WebSocket-aware)                                        |

---

## Architecture

```
workplace-app/
├── cmd/server/main.go          # Entry point — wires all dependencies and starts the server
├── internal/                   # Domain packages (one per feature)
│   ├── asset/
│   ├── attendance/
│   ├── auth/
│   ├── chat/
│   ├── claim/
│   ├── department/
│   ├── leave/
│   ├── memo/
│   ├── milestone/
│   ├── notification/
│   ├── overtime/
│   ├── project/
│   ├── projectcomment/
│   ├── query/
│   ├── reminder/
│   ├── report/
│   ├── review/
│   ├── task/
│   └── user/
├── pkg/                        # Shared infrastructure
│   ├── cache/                  # Redis helpers
│   ├── config/                 # Environment config
│   ├── database/               # DB connection setup
│   ├── middleware/             # Auth, CORS, logging
│   ├── response/               # HTTP response helpers
│   └── utils/                  # Geofencing, JWT, hashing, UUIDs
├── migrations/                 # SQL migration files (run in order)
├── docker/                     # Nginx config
├── docker-compose.yml
└── Dockerfile
```

Each domain package follows the same structure:

```
<domain>/
├── model.go        # Structs and types
├── dto.go          # Request/response data transfer objects
├── repository.go   # Database queries (pgx)
├── service.go      # Business logic
└── handler.go      # HTTP handlers and route registration
```

---

## Getting Started

### Prerequisites

- [Docker](https://www.docker.com/) and Docker Compose
- Go 1.22+ (only needed for local development outside Docker)

### Clone and configure

```bash
git clone <repo-url>
cd workplace-app
cp .env.example .env   # or edit .env directly
```

---

## Running the Project

### With Docker (recommended)

```bash
# Build images
docker-compose build

# Start all services (app, postgres, redis, nginx, mailhog)
docker-compose up -d

# Stop all services
docker-compose down
```

Services after startup:

| Service                | URL                   |
| ---------------------- | --------------------- |
| API (via Nginx)        | http://localhost:8090 |
| API (direct)           | http://localhost:5002 |
| MailHog (dev email UI) | http://localhost:8025 |
| PostgreSQL             | localhost:5555        |
| Redis                  | localhost:6500        |

### Local development (without Docker)

```bash
# Ensure PostgreSQL and Redis are running locally, then:
go run ./cmd/server/main.go
```

---

## API Overview

All routes are prefixed with `/api/v1`. Protected routes require a JWT token in the `Authorization: Bearer <token>` header or in a cookie.

```
POST /api/v1/auth/login
POST /api/v1/auth/register

GET|POST        /api/v1/users
GET|PUT|DELETE  /api/v1/users/:id

GET|POST        /api/v1/departments
GET|PUT|DELETE  /api/v1/departments/:id

GET|POST        /api/v1/projects
GET|PUT|DELETE  /api/v1/projects/:id

GET|POST        /api/v1/tasks
GET|POST        /api/v1/milestones
GET|POST        /api/v1/comments

POST            /api/v1/attendance/sign-in
POST            /api/v1/attendance/sign-out
PUT             /api/v1/attendance/:id/sign-out     # Admin override
GET             /api/v1/attendance/
GET             /api/v1/attendance/me

GET|POST        /api/v1/overtime
GET|POST        /api/v1/memos
POST            /api/v1/memos/:id/acknowledge
GET             /api/v1/memos/:id/acknowledgements

GET|POST        /api/v1/queries
GET|POST        /api/v1/leaves
GET|POST        /api/v1/reviews
GET|POST        /api/v1/claims
GET|POST        /api/v1/assets

GET             /api/v1/notifications
GET             /api/v1/reports

WS              /api/v1/chat/ws
WS              /api/v1/notifications/ws
```

---

## Modules

| Module             | Description                                                                 |
| ------------------ | --------------------------------------------------------------------------- |
| **auth**           | Login, registration, JWT issuance and validation                            |
| **user**           | User profiles, roles, department assignment                                 |
| **department**     | Department creation and membership                                          |
| **project**        | Projects with status, budget, file attachments, and history                 |
| **task**           | Tasks with assignment, progress, and due dates                              |
| **milestone**      | Project milestones and completion tracking                                  |
| **projectcomment** | Threaded discussion on projects                                             |
| **attendance**     | Sign-in/out with GPS geofencing, device lock, and admin override            |
| **overtime**       | Overtime submissions, approval, and payment tracking                        |
| **memo**           | Company-wide, department, or individual memos with per-user acknowledgement |
| **query**          | Employee queries directed to HR or management                               |
| **leave**          | Leave requests with approval workflow                                       |
| **review**         | Performance reviews between managers and employees                          |
| **claim**          | Expense claims with approval and payment status                             |
| **asset**          | Company asset inventory and assignment to employees                         |
| **chat**           | Real-time WebSocket messaging (DMs and rooms)                               |
| **notification**   | In-app and email notifications; real-time push via WebSocket                |
| **report**         | Aggregated analytics on attendance, overtime, projects, and HR data         |
| **reminder**       | Background scheduler for automated daily jobs                               |

---

## Database Migrations

Run migrations in order. Each file is idempotent where possible (`IF NOT EXISTS`).

| File                              | Description                                                               |
| --------------------------------- | ------------------------------------------------------------------------- |
| `001_schema.sql`                  | Core schema: all primary tables and relationships                         |
| `002_seed.sql`                    | Default super admin user                                                  |
| `003_add_user_role.sql`           | Adds `user` role to enum                                                  |
| `004_user_position_dept_name.sql` | Adds `position` and `department_name` columns to users                    |
| `005_new_features.sql`            | Adds tasks, milestones, comments, project history, budget, files, queries |
| `006_remove_user_role.sql`        | Removes `user` role; migrates existing users to `staff`                   |
| `007_add_procurement_role.sql`    | Adds `procurement` role                                                   |
| `008_memo_acknowledgements.sql`   | Per-user memo acknowledgement junction table                              |
| `009_attendance_auto_closed.sql`  | Adds `auto_closed` flag to attendance records                             |
| `010_attendance_device_id.sql`    | Adds `device_id` column for buddy-punch prevention                        |

To apply a migration against the running Docker container:

```bash
docker-compose exec postgres psql -U postgres -d workplace -c "<SQL statement>"
```

---

## Background Jobs

The `reminder` package runs three scheduled jobs daily:

| Time  | Job                   | Description                                                                                |
| ----- | --------------------- | ------------------------------------------------------------------------------------------ |
| 08:00 | Deadline reminders    | Notifies assignees of projects and tasks due the next day                                  |
| 16:30 | Sign-out reminder     | Pushes a notification to every employee still signed in without a sign-out                 |
| 18:00 | Auto-close attendance | Closes all open attendance records; sets `auto_closed = true`; notifies affected employees |

---

## Authentication

- Tokens are signed JWTs containing `user_id`, `role`, and `department_id`
- Active tokens are cached in Redis; invalidation clears the cache entry
- Role values: `super_admin`, `dept_head`, `staff`, `procurement`
- WebSocket connections authenticate via `?token=<jwt>` query parameter

---

## WebSockets

### Chat — `WS /api/v1/chat/ws`

Handles real-time messaging. Message types: `message`, `typing`, `reaction`, `delete`.

### Notifications — `WS /api/v1/notifications/ws`

Pushes notification payloads to connected clients in real time. Falls back to polling every 30 seconds on the frontend if the WebSocket drops.

Both WebSocket endpoints reconnect automatically on the client side with a 5-second delay.

---

## Geofencing & Device Lock

### Geofencing

Attendance sign-in checks the submitted GPS coordinates against the configured company location using the Haversine formula. If the employee is outside `GEOFENCE_RADIUS` metres, sign-in is rejected.

Configure the company location in `.env`:

```env
COMPANY_LAT=6.62501
COMPANY_LNG=3.34346
GEOFENCE_RADIUS=100
```

### Device Lock (production only)

When `ENV=production`, each browser is assigned a persistent UUID stored in `localStorage`. If a device has already been used to sign in a different employee today, the sign-in is rejected. This prevents buddy-punching.

The check is skipped entirely when `ENV=development`, so local testing with multiple accounts is unaffected.

---

## Makefile Commands

```bash
make up         # Start Docker Compose stack
make down       # Stop and remove containers
make build      # Build Docker images
make run        # Run server locally (go run)
make tidy       # go mod tidy
make lint       # Run golangci-lint
make test       # Run all tests
make seed       # Apply seed migration (002)
```

---
