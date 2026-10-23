# LONG Hub

## Deploy

Before deploying, a Cloudflare Worker is required to send outbound requests from Webhooks.

```sh
$ cd worker
$ npm install
$ npx wrangler deploy -e=prod   # change in local
$ npx wrangler secret put SERVER_PUBLIC_KEY -e=prod
# paste webhook public key, see below
```

To generate keypairs (including webhook key and jwt key), run `cmd/setup`:
```sh
$ cd backend
$ go run ./cmd/setup
```

A `.env` file will be created from `.env.template` with randomly generated keys.

### Docker

```sh
$ cp .env.template .env
$ vim .env
$ docker compose up -d
```

Frontend port is mapped to localhost:5173 and backend port is mapped to localhost:8000.

### Manual

When deploying manually or setting up for development, `.env` files are required respectively for frontend and backend, as when in Docker it is loaded by Docker Compose from the root directory.

Requirements:
- Go >= 1.26
- Node.js >= 25 with corepack enabled
- [goose](https://github.com/pressly/goose)
- PostgreSQL


Extra environment variables in `backend/.env`:
```
GOOSE_DRIVER=postgres
GOOSE_DBSTRING=postgres://user:password@host/db  # replace this
GOOSE_MIGRATION_DIR=./db/migrations
```

```sh
$ git clone https://github.com/MoveToEx/longhub.git
$ cd longhub
$ cd frontend
$ cp .env.template .env
$ vim .env
$ yarn
$ yarn dev

### (in another terminal)
$ cd backend
$ go run ./cmd/setup
$ vim .env
$ goose up
$ go mod download
$ go build
$ ./long
```

## Admin panel

The **Admin** sidebar entry opens `/admin/images` and `/admin/users`. It is visible
only to signed-in users with the existing Admin permission (`8`). Both pages are
guarded when opened directly. Deletion requests have a separate moderation tab.
Permissions use four bits:
Create (`1`), Edit (`2`), Request deletion (`4`), and Admin (`8`). The Users table
displays them in `ADEC` order, using `-` for missing bits (for example, `--EC`).

An operator can grant Admin to an existing account in PostgreSQL, preserving its
other permissions:

```sql
UPDATE public."user" SET permission = permission | 8 WHERE username = 'your-username';
-- Revoke Admin without removing other permissions:
UPDATE public."user" SET permission = permission & ~8::BIGINT WHERE username = 'your-username';
```

Reload the frontend after granting access. Admin requests check the account's
current database permissions, so revocation takes effect on the next request,
even for an existing session. App keys require Admin on both the key and its owner.

### GraphQL API

Send JSON requests to `POST /admin/graphql` with `Content-Type: application/json`
and either `Authorization: Session <token>` or `Authorization: Key <app-key>`.
Invalid authentication returns HTTP 401; insufficient permissions return 403.
Authorized requests use the GraphQL `data` / `errors` response format.

```graphql
query InspectImages($input: ImageQueryInput) {
  images(input: $input) {
    total
    items {
      id imageUrl imageKey createdAt updatedAt deletedAt
      uploader { id username }
      currentVersion { id version text rating tags }
    }
  }
}
```

Example variables:

```json
{"input":{"status":"ALL","search":"featured","offset":0,"limit":25,"sortBy":"CREATED_AT","direction":"DESC"}}
```

```graphql
query InspectUsers {
  users(input: { search: "example.com", limit: 25, sortBy: USERNAME, direction: ASC }) {
    total
    items { id username email permission createdAt activeImageCount versionCount }
  }
}
```

Pagination defaults to offset `0`, limit `25`, and creation time descending, with
an ID tie-breaker. Limits must be between `1` and `100`. Image status defaults to
`ACTIVE`; `DELETED` and `ALL` include deleted records. A missing current version
is returned as `null`. Image sorting accepts `ID`, `CREATED_AT`, and `UPDATED_AT`;
user sorting accepts `ID`, `USERNAME`, `EMAIL`, `CREATED_AT`, and `PERMISSION`.

Search matches exact numeric IDs or case-insensitive substrings of image text,
uploader names, and current-version tags, or usernames/emails on the Users page.
SQL wildcard characters are treated literally. Searches are limited to 255 UTF-8
bytes. IDs are strings, timestamps are ISO 8601, and `Int64` counts/permission masks
are JSON numbers. User image counts include active images; version counts include
all versions. Credentials, handles, preferences, and integration secrets are not
part of the admin schema.

Responses use `Cache-Control: no-store`. The endpoint accepts JSON POST only,
limits request bodies to 1 MiB and query complexity to 5000 (including list size),
and gives database queries a 10-second deadline. The only mutation is deletion
moderation; no subscriptions or playground are exposed.

### Deletion requests

Image pages show public, paginated request history in a collapsed panel. Creating
a request requires the existing **D bit (4)** specifically; Admin alone does not
qualify. An application key needs D on both the key and its owner. Permissions are
read from the database for each request, so revocation takes effect immediately.
Existing bit values and registration defaults are unchanged.

`GET /image/:id/deletion?offset=0&limit=25` returns `{data:{items,total}}`, including
reason, requester ID/username, status, creation time and processing time. The limit
is 1–100. `POST /image/:id/deletion` accepts `{ "reason": "Duplicate of …" }` with
Session or Key authentication. Reasons are trimmed and must contain 1–2,000 Unicode
characters. Each user may have one Active request per image. Duplicate active
requests and submissions for deleted images return HTTP 409; missing images
return 404. Full history stays public after deletion.

Admins moderate at `/admin/deletions` or through GraphQL:

```graphql
query ReviewDeletionRequests {
  deletionRequests(input: { status: ACTIVE, limit: 25, sortBy: CREATED_AT, direction: DESC }) {
    total
    items {
      id imageId reason status createdAt processedAt imageDeletedAt activeDeletionId
      requester { id username }
      moderator { id username }
    }
  }
}

mutation AcceptRequest {
  moderateDeletionRequest(id: "123", action: ACCEPT, expectedStatus: ACTIVE) {
    id status imageDeletedAt activeDeletionId
  }
}
```

Filters support `ACTIVE` (default), `APPROVED`, `REJECTED`, `REVOKED`, and `ALL`.
Search matches exact request/image IDs, requester names, or reasons. Sorting
supports `ID`, `CREATED_AT`, and `PROCESSED_AT`; pagination uses the existing limits.
Active requests support Accept/Reject; Approved supports Revoke; Rejected and
Revoked support Reopen/Accept. Send the status last seen as `expectedStatus`.
Stale or invalid transitions return a GraphQL error with `extensions.code: CONFLICT`.
Only one request may control deletion at a time. Reopening clears moderator and
processing time; all decisions record the acting moderator.

Approval hides an image from discovery, user lists, favorites, recommendations,
and search. Direct details and version/request history remain available, including
the original image and deletion reason/date. Metadata edits and new favorites are
blocked while deleted. Revocation restores visibility and retained favorites.
Files, metadata, and existing favorite records are preserved.

The deletion migration adds moderator attribution, uniqueness/index constraints,
and a transactional outbox. Search changes and deletion webhook jobs are dispatched
after commit, with a periodic retry every 10 seconds. Search workers serialize with
moderation and use current database state; database filtering hides deleted hits
immediately, while index totals converge after synchronization. Each approval
captures an immutable image snapshot and the approving admin as initiator. Its
`data.deletionEventId` identifies the approval, including reapproval after revocation.
Webhook delivery retains the existing retry/failure handling. Restoration emits no
webhook. Request editing, withdrawal, and hard deletion are not supported.

### Development and verification

The GraphQL schema is in `backend/internal/admin/schema.graphqls`. After changing
it, run `go generate ./internal/admin` from `backend`; the generator is pinned as
a Go tool. After changing SQL, run `sqlc generate` there too.

Run `go test ./...` and `sqlc vet` in `backend`, and `yarn test` and `yarn build` in
`frontend`. PostgreSQL integration tests require `TEST_DATABASE_URL` pointing to
a local PostgreSQL account allowed to create databases. Tests create, migrate,
and remove their own temporary databases; the named source database is not changed.

For dependency downloads in this environment, use `http://localhost:7890` as both
the HTTP and HTTPS proxy. Yarn supports `YARN_HTTP_PROXY` / `YARN_HTTPS_PROXY`;
Go supports `HTTP_PROXY` / `HTTPS_PROXY`. Keep certificate verification enabled.

# TODO

- [x] Admin panels based on GraphQL
- [ ]
