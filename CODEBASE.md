# ArcVault 2.0 Codebase Primer

**Project Overview:**
ArcVault 2.0 is an enterprise-grade, distributed job orchestration and execution platform with three main components:
- **Coordinator** (Go): Central management server with SQLite database, HTTP API, WebSocket hub
- **Agent** (Go): Lightweight distributed job executor on target machines
- **Dashboard** (Vue.js/TypeScript): Web UI for job creation, monitoring, and administration

## Core Architecture

The system operates on a polling + WebSocket hybrid model:
1. Agents register with the coordinator via heartbeat (every 30s)
2. Agents poll coordinator for pending jobs (every 30s)
3. Agents execute jobs (robocopy on Windows, rsync on Unix)
4. WebSocket maintains persistent bidirectional connection for commands and progress updates
5. Real-time WebSocket broadcasts job status to all connected dashboards

## Key Features

### Job Management
- Single-agent or group dispatch (one job per group member, shared dispatch_id)
- Cron-based scheduling (5-field expressions)
- Status tracking: pending → running → completed/failed
- Progress reporting (files processed, bytes transferred)
- Sync flags for customizing robocopy/rsync behavior
- Job result storage with error messages

### Credential System
- Encrypted credential storage (AES-256-GCM in SQLite)
- Type-specific credential application:
  - SMB: cmdkey on Windows
  - SSH: temp key files or sshpass environment variable
- Reactive credential filtering in job creation UI
- Credential profiles assignable to jobs

### Federation
- Multi-site deployment with root/subscriber architecture
- Federation hub for gossip-based synchronization
- Offline detection and health monitoring
- Lag events tracking (max 15s polling interval)
- Federation messages for state replication

### Authentication & Authorization
- JWT tokens (4-hour TTL) with HS256 signing
- Role hierarchy: admin > operator > viewer
- Fallback to legacy admin/agent tokens for backward compatibility
- Password-change enforcement on first login
- Role-based access control (RBAC) on routes

### Notifications
- Multiple delivery channels: Webhook, Email (SMTP), Slack, Teams
- Triggered on job failure (configurable)
- Job failure context: name, agent, run ID, duration, error message
- Retry logic with exponential backoff

### Progress Tracking
- Real-time parsing of robocopy (per-file %) and rsync (overall %)
- Custom scanner handling \r\n\r splits for in-place progress
- Streaming progress to UI via WebSocket broadcast

## Server Components

### Hub (WebSocket)
- Client broadcast pool (dashboard connections)
- Agent connection map (indexed by agent ID)
- Async writes with 5s deadline to prevent stalls
- Dead client cleanup

### Business Services
- **AgentService**: register, heartbeat, list, delete (with running job check)
- **JobService**: CRUD, group dispatch, list with filters, pagination
- **UserService**: create, list, password change, bcrypt hashing
- **GroupService**: create, list, add/remove members

### Database (SQLite)
- WAL mode for concurrent readers + single writer
- 5s busy timeout for write queueing
- Tables: agents, jobs, job_runs, users, groups, credentials, federation, tokens, alert_rules
- Cascade deletes for cleanup

## Agent Execution Flow

1. **Agent config** (YAML): agent_id, coordinator_url, auth_token, ca_cert
2. **On startup**:
   - Register via POST /api/agents/register
   - Start heartbeat loop (30s intervals)
   - Start job runner poll loop (30s intervals)
   - Start WebSocket client with failover coordinators
3. **For each pending job**:
   - Update status to running
   - Apply credentials (SMB/SSH)
   - Execute robocopy/rsync with progress parsing
   - Post result (exit code, output)
   - Update status to completed/failed
4. **Cleanup** credentials after job

## Key Patterns

- **DTOs** for API contracts (JobDTO, AgentDTO, etc.) — clean separation of domain models
- **Middleware chain** for auth, role checks, password change enforcement
- **Interface-based DB** (AgentQueries, JobQueries, etc.) for testability
- **Async broadcast** with lock snapshots to prevent write stalls
- **Cron entry map** for dynamic job scheduling add/remove

## Notable Implementation Details

- TLS self-signed cert generation on init (32-byte encryption key also generated)
- Admin token fallback for backward compatibility (no JWT required)
- Bootstrap tokens expire after 1 hour (stored in tokens table)
- Robocopy buffer: 256KB scanner for wide lines
- No per-file progress in robocopy (/ NP explicitly omitted)
- Rsync uses --info=progress2 for single overall % line
- Federation lag tracked via periodic health polls (15s intervals)
- Offline agent detection (90s threshold after last heartbeat)
- Alert history retention configurable (default days not specified in code)

## Security Considerations

- Credentials encrypted before storage, decrypted at job execution time
- JWT JTI-based revocation (logout tokens)
- SQL injection prevention via parameterized queries
- Bcrypt for user passwords (DefaultCost)
- TLS mandatory for agent → coordinator (except localhost for dev)

## Project Structure

```
.
├── coordinator/          # Backend server (Go)
│   ├── cmd/             # CLI commands (init, start, rekey)
│   ├── config/          # Configuration loading
│   ├── db/              # Database layer
│   ├── business/        # Business services (agents, jobs, users, groups)
│   ├── server/          # HTTP handlers & WebSocket hub
│   ├── notifications/   # Email, Slack, Teams, Webhook
│   ├── internal/        # Credential crypto, TLS cert generation
│   └── main.go          # Entry point
├── agent/               # Client agent (Go)
│   ├── config/          # Agent configuration
│   ├── heartbeat/       # Registration & heartbeat
│   ├── runner/          # Job execution (robocopy/rsync)
│   ├── service/         # Windows service integration
│   ├── ws/              # WebSocket client
│   ├── updater/         # Agent update logic
│   └── main.go          # Entry point
└── dashboard/           # Web UI (Vue.js/TypeScript)
    ├── src/
    │   ├── api.ts       # API client
    │   ├── main.js      # Vue app entry
    │   ├── router/      # Route definitions
    │   ├── views/       # Page components
    │   ├── composables/ # Reusable logic (auth, websocket, federation lag)
    │   ├── schemas/     # Zod validation schemas
    │   ├── types/       # TypeScript interfaces
    │   └── utils/       # Helpers (cron, formatting)
    └── vite.config.js   # Build config
```

## Development Notes

### Adding a New Endpoint
1. Define handler in `coordinator/server/*.go`
2. Register route in `server.registerRoutes()`
3. Add business logic in `coordinator/business/*.go`
4. Add DB queries in `coordinator/db/*.go`
5. Add API client in `dashboard/src/api.ts`
6. Add schema validation in `dashboard/src/schemas/*.ts`
7. Update dashboard views/components

### Adding a New Feature
1. Design the database schema and migrations
2. Implement DB layer queries
3. Implement business service logic with DTOs
4. Implement HTTP handlers with middleware
5. Wire up WebSocket broadcasts if needed
6. Implement API client and validation schemas
7. Build dashboard UI with composables for state management

### Testing
- Business logic: `*_test.go` files (mocks in `mocks_test.go`)
- Frontend: `.test.js` files with Jest
- Integration: `cmd/arcvault-test/` for end-to-end scenarios

## API Contract Sync Points

Dashboard and coordinator share type contracts. Key sync points (with last sync dates in comments):
- `dashboard/src/types/api.ts` mirrors `coordinator/server/*_types.go`
- Zod schemas in `dashboard/src/schemas/` validate coordinator responses
- Version: Last synced 2026-06-03

## Future Enhancements & Known Considerations

- Alert history retention days currently not enforced (configurable in code but no default)
- Missed schedule alerts only fire if no recent alert exists (dedup logic)
- Federation lag detection relies on 15s polling (no real-time push)
- Credential key rotation via `coordinator rekey` command (no automatic rotation)
- Agent rollback capability tracked but not fully implemented in UI
