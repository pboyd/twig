<img src="assets/logo.svg" alt="twig logo" width="120" align="right">

# twig

Twig is a terminal-first to do list with the following features:

* Self-hosted server
* Full-featured TUI
* Mobile-optimized web app
* CLI interface
* Tasks to keep track of stuff you need to do
* Plans to organize your day
* Goals to keep bigger projects on track
* Pomodoro timer to keep you focused
* Markdown rendering almost everywhere
* Text is editable in your `$EDITOR`

## Quick start

Requirements: `podman` and `podman-compose`, or Docker with the Compose plugin.

1. **Save this as `compose.yaml`:**

   ```yaml
   services:
     postgres:
       image: docker.io/library/postgres:17-alpine
       environment:
         POSTGRES_USER: twig
         POSTGRES_PASSWORD: twig
         POSTGRES_DB: twig
       healthcheck:
         test: ["CMD-SHELL", "pg_isready -U twig"]
         interval: 5s
         timeout: 5s
         retries: 5
       volumes:
         - postgres_data:/var/lib/postgresql/data
       restart: unless-stopped

     server:
       image: ghcr.io/pboyd/twig-server:latest
       environment:
         DATABASE_URL: postgres://twig:twig@postgres:5432/twig?sslmode=disable
       ports:
         - "8080:8080"
       depends_on:
         postgres:
           condition: service_healthy
       restart: unless-stopped

   volumes:
     postgres_data:
   ```

   Pin `:latest` to a `vX.Y.Z` tag once you care about reproducibility, and change the `twig`/`twig`
   credentials if this isn't just your laptop.

2. **Start it:**

   ```bash
   podman-compose up -d
   # or: docker compose up -d
   ```

   Migrations run automatically on startup. The server is up at `http://localhost:8080`.

3. **Provision a user** (prints an API key — save it, it can't be recovered):

   ```bash
   podman-compose exec server /server --provision-user alice:secret
   # or: docker compose exec server /server --provision-user alice:secret
   ```

   `/server` must be spelled out — the image is distroless, so there's no shell and no `PATH`.

4. **Open the web app:** `http://localhost:8080`, and log in as `alice`.

5. **Get the CLI:** the server ships a matching linux/amd64 binary. Log in, open the
   **Download** page (`http://localhost:8080/download`), and download it — the endpoint is
   authenticated, so this is easier via the browser (already holding your session) than `curl`. If
   you already have Go 1.26+, `go install github.com/pboyd/twig/cmd/twig@latest` also works.

   ```bash
   chmod +x twig
   export TWIG_API_KEY=<key from step 3>
   export TWIG_ADDR=http://localhost:8080   # the default
   ./twig            # interactive TUI
   ./twig task list  # or use subcommands: goal, task, pom, plan, report
   ```

### Developing on twig

Building from source instead of using the published image:

```bash
git clone https://github.com/pboyd/twig && cd twig
make dev          # builds services/twig/Dockerfile, starts it with postgres on :8080
make cli          # builds ./twig from source (needs Go 1.26+)
```

The web app's own dev server (hot reload, proxies API calls to `:8080`):

```bash
cd services/twig-web
npm install
npm run dev   # → http://localhost:5173
```

Log in with a name and password provisioned as in Quick start step 3. See `CLAUDE.md` / `AGENTS.md`
for the rest of the dev workflow.

### Deploy to a real host

The `deploy/` directory contains a self-contained Ansible example that turns a bare Fedora host into
a working Twig instance — two roles, four configuration values, and a first run that needs no
external service. See [`deploy/README.md`](deploy/README.md) for the walkthrough.
