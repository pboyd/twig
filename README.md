<img src="assets/logo.svg" alt="twig logo" width="120" align="right">

# twig

Twig is a personal task tracker built around goals, daily plans, and Pomodoro sessions. It ships as three pieces:

- **CLI/TUI** (`cmd/twig`) — manage goals and tasks, run Pomodoro timers, build a daily plan, and review activity reports. Running `twig` with no arguments opens an interactive TUI.
- **Server** (`services/twig`) — an HTTP/2 ConnectRPC API backed by PostgreSQL.
- **Web app** (`services/twig-web`) — a React SPA over the same API.

## Quick start

Requirements: Go 1.26+, `podman` and `podman-compose`.

1. **Start the server and database:**

   ```bash
   make dev
   ```

   This builds the server image and starts it with PostgreSQL on `localhost:8080`. Migrations run automatically.

2. **Provision a user** (prints an API key):

   ```bash
   podman-compose exec server /server --provision-user alice:secret
   ```

3. **Build and configure the CLI:**

   ```bash
   make cli
   export TWIG_API_KEY=<key from step 2>
   export TWIG_ADDR=http://localhost:8080   # the default
   ```

4. **Run it:**

   ```bash
   ./twig            # interactive TUI
   ./twig task list  # or use subcommands: goal, task, pom, plan, report
   ```

### Web app (optional)

```bash
cd services/twig-web
npm install
npm run dev   # → http://localhost:5173, proxies API calls to :8080
```

Log in with the name and password from step 2.

### Deploy to a real host

The `deploy/` directory contains a self-contained Ansible example that turns a bare Fedora host into
a working Twig instance — two roles, four configuration values, and a first run that needs no
external service. See [`deploy/README.md`](deploy/README.md) for the walkthrough.
