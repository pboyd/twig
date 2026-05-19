# Quickstart: User Authentication

How to apply the auth migration, provision users, and exercise both credential
types. Assumes the `services/todo` working directory and a reachable PostgreSQL
instance in `DATABASE_URL`.

## 1. Build

```sh
cd services/todo
go build ./...                       # server + todo CLI
```

The auth migration `000003_auth` runs automatically on server startup (the
server calls `migrate.Up()` before serving).

## 2. Provision users

Run the server once with one `--provision-user` flag per account. The server
applies migrations, creates each account, issues one API key per account,
prints each raw key, and exits.

```sh
DATABASE_URL=postgres://... ./server \
  --provision-user=alice:s3cr3t \
  --provision-user=bob:hunter2
```

Example output:

```
migrations applied
provisioned user "alice" — API key: 7f3a...e12c
provisioned user "bob"   — API key: c90b...44a1
```

Record each key — it is shown only once. Provisioning an existing username
resets that user's password and issues a fresh key (the old key stops working).

## 3. Start the server normally

```sh
DATABASE_URL=postgres://... ./server      # no --provision-user flags
```

The server listens on `:8080`. All task RPCs now require a credential.

## 4. CLI client — API key

The `todo` CLI authenticates with an API key from the `TODO_API_KEY`
environment variable:

```sh
export TODO_ADDR=http://localhost:8080
export TODO_API_KEY=7f3a...e12c          # alice's key

todo task add "write the auth tests"
todo task list                           # shows only alice's tasks
```

With `bob`'s key exported instead, `todo task list` shows only `bob`'s tasks.
An unset or wrong `TODO_API_KEY` makes every task command fail with `401`.

## 5. Browser-style client — session login

Sign in to obtain a session cookie, then call task RPCs with that cookie:

```sh
# log in — capture the Set-Cookie value
curl -i -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"s3cr3t"}'
# → 200 OK, Set-Cookie: todo_session=<id>; HttpOnly; Secure; SameSite=Strict

# subsequent task RPCs carry the cookie instead of a Bearer token
curl -i http://localhost:8080/task.v1.TaskService/ListTasks \
  -H 'Content-Type: application/json' \
  -H 'Cookie: todo_session=<id>' \
  -d '{}'

# log out — the session is deleted
curl -i -X POST http://localhost:8080/auth/logout \
  -H 'Cookie: todo_session=<id>'
```

A wrong password returns `401 {"error":"invalid username or password"}` with no
cookie. After logout, the old cookie returns `401` on task RPCs.

## 6. Verify isolation

```sh
# as alice
TODO_API_KEY=<alice-key> todo task add "alice only"
# as bob — does not see alice's task
TODO_API_KEY=<bob-key>   todo task list
```

Fetching another user's task by its numeric id returns `not found`, never the
task.

## 7. Run the tests

```sh
cd services/todo
go test ./...
```

Covers credential helpers (bcrypt, token generation, SHA-256, constant-time
compare), the auth middleware, the `/auth/login` and `/auth/logout` endpoints,
and per-user task isolation in the task handlers. No live PostgreSQL is needed
for the unit and middleware tests.
