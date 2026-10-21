# f0rum

Server-rendered Go/SQLite forum. Requires Go 1.20+ and a working CGO C toolchain for go-sqlite3. Production must terminate HTTPS at a trusted reverse proxy; secure cookies are enabled by default.

## Local development

From the project directory:

```
go run ./app -root . -insecure-local
```

The default listener is 127.0.0.1:8080. -root locates templates/css/database; -db selects an explicit database path. Never expose -insecure-local outside local development.

## Existing databases

Legacy databases are never silently changed at startup. Stop the server and rehearse on a database copy first:

```
go run ./app -root . -db path/to/copy.db -migrate
```

Migration writes a timestamped SQLite backup, validates unique identities and relationships, migrates transactionally, and invalidates old sessions. Duplicate/orphan/conflicting data causes rollback. Legacy comment-vote direction was not recorded; its historical counters are preserved rather than fabricated.

To assign administrator status, use -migrate -admin-user-id N only after verifying the existing numeric user ID. Names, including Deus, grant no privileges. No administrator is created automatically.

## Checks

```
go test ./...
CGO_ENABLED=1 go test ./app/forum
```

The CGO integration suite uses disposable databases for account uniqueness, authorization, forged author fields, atomic writes, voting, expiry and cascades. Without CGO only the driver-independent checks run.

## Container

The image builds CGO explicitly, keeps user data out of the image, and runs as an unprivileged user. Mount a writable /data volume, expose the application only behind HTTPS, and supply the intended listener/database arguments. Backups and migrations should be run while the application is stopped.

## Continuous integration and regression tests

Use Go 1.26 or later, SQLite's CGO driver and a C compiler (GCC on Linux):

```sh
CGO_ENABLED=1 go test -race -count=1 -coverpkg=./... -coverprofile=coverage.out ./...
go vet ./...
python app/tests_schema.py
```

CI uses Go 1.27 on Ubuntu with CGO enabled. It builds the server, runs the real
SQLite HTTP/store integration tests and unit tests with race detection, and uploads
test logs and coverage. The minimum total statement coverage is recorded in
`.github/coverage-baseline.json`. SQLite tests use temporary databases; authentication,
session rotation/revocation, CSRF, role enforcement, atomic writes, vote uniqueness,
query filters, escaping and schema-version rejection are exercised without live data.
Synthetic legacy migration tests also verify backup contents, identity preservation,
invalidated legacy sessions and complete rollback on orphan or duplicate records.
The hosted suite currently reports 76.67% total statement coverage, including the
server entry point, with a 76% regression floor.
The Python migration rehearsal reads the existing fixture database and changes only
a disposable copy; it is an additional local check, not a production migration.

CodeQL and weekly Go/Actions Dependabot updates run through `.github`. The proposed
branch ruleset requires both CI jobs; it must be applied separately in GitHub settings.
