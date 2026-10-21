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
