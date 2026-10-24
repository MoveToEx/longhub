# Replicating the image catalog

Use a full export to bootstrap or replace a local catalog, then use webhooks as
notifications to refresh individual images. This protocol uses the existing
metadata version IDs. It adds no database tables, columns, or revision counters.

The replica contains current image metadata and remote image URLs. It does not
contain image files, historical metadata versions, favorites, or account secrets.

## Download a full export

Send `GET /sync/full` with either `Authorization: Key <app-key>` or
`Authorization: Session <session-token>`. Any authenticated user or app key can
read the export, including an app key with zero write permissions.

```sh
curl --fail --compressed \
  -H "Authorization: Key $LONGHUB_APP_KEY" \
  https://api.example.com/sync/full --output staging.ndjson
```

The response is `application/x-ndjson`, with `Cache-Control: no-store`. It is
gzip-compressed when the request accepts gzip. HTTP errors before streaming use
the normal JSON error response. An error after streaming starts aborts the export
without a successful end record, even if the HTTP status is already 200.

Records appear in this order:

1. A manifest: `{"type":"manifest","schemaVersion":1,"exportId":"…","startedAt":"…"}`.
2. Zero or more `{"type":"image","data":{…}}` records, ordered by ascending
   image ID. `data` has the same shape as the `data` returned by `GET /image/:id`:
   `id`, `text`, `rating`, `imageUrl`, `imageKey`, `createdAt`, `userIdentifier`
   (`id`, `createdAt`, `username`), `currentVersionId`, `version`, `tags`
   (`[{"id":1,"name":"example"}]`), and `deletion` (null for exported images).
3. An end record: `{"type":"end","exportId":"…","recordCount":123,"sha256":"…"}`.

Each record ends with one LF byte. The SHA-256 is the lowercase hexadecimal digest
of the exact **uncompressed image record lines, including their LF bytes**. Exclude
the manifest and end record. Validate the schema version, matching export IDs,
record count, checksum, end record, and successful EOF (including gzip integrity)
before accepting the download. Unknown additive object fields may be ignored.

All image metadata and tags come from one read-only, repeatable-read PostgreSQL
snapshot. Deleted images and images without a current metadata version are
excluded. A refresh must **replace** the previous catalog, rather than only
upserting returned images, so that absent images disappear locally.

Downloads cannot resume. Discard incomplete staging data and start another export.
`startedAt` and `exportId` describe this download; neither is a change-feed cursor.
IDs are JSON integers, as in the existing API; clients must preserve 64-bit values
(for example, use a lossless JSON parser in JavaScript).

The standard-library Python [download example](./examples/)
validates the stream and atomically replaces a local file only after success:

```sh
export LONGHUB_APP_KEY='long-your-app-key'
python docs/examples/fetch_catalog.py https://api.example.com staging-images.ndjson
```

Its output contains only image record lines. This example handles download and
validation; integrate the webhook handoff below before publishing an application's
new local catalog.

## Webhooks are refresh notifications

Register an active webhook with `eventTypes: 7`: creation (1), update (2), and
deletion/restoration (4). The callback must be publicly reachable; the Cloudflare
gateway rejects localhost and private addresses.

The existing deletion subscription now includes both `"delete"` and `"restore"`
events. Restoration reuses the existing deletion outbox's JSON payload and
subscription bit. Older queued deletion payloads without an explicit event name
continue to produce `"delete"`.

```json
{
  "event": "restore",
  "data": {
    "imageID": 42,
    "imageURL": "https://images.example.com/42.png",
    "text": "Example caption",
    "rating": "none",
    "tags": ["example"],
    "createdAt": "2026-10-24T00:00:00Z",
    "uploader": { "id": 7, "username": "contributor" },
    "deletionEventId": "deletion-123"
  },
  "initiator": { "id": 9, "username": "moderator" }
}
```

Use the default envelope or a custom template preserving `.Event` and
`.Data.ImageID`. Update existing receivers and templates to handle `restore`
before deploying the backend change. Deletion event IDs identify notifications;
they do not establish ordering. The payload's `createdAt` is the image creation
time, not the time of deletion or restoration.

For **every** create, update, delete, or restore notification:

1. Verify `X-Signature` against the exact raw body using HMAC-SHA256 and your
   webhook secret. Durably queue the image ID before returning 2xx.
2. Fetch `GET /image/:id` and read its normal JSON `data` object.
3. Remove the local image when `data.deletion` is non-null, `currentVersionId` is
   null, or the endpoint returns 404. Otherwise upsert the returned image.
4. Retry transient HTTP failures; do not treat them as deletion.

Serialize refreshes for each image. Notifications arriving while a refresh is in
flight must remain queued and cause another read. Acknowledge only the queue
receipts claimed for that read, not every notification for the image. This avoids
losing an update that arrives between reading the server and writing locally.

Always refresh deletion/restoration notifications even when `currentVersionId`
has not changed. A metadata version identifies text/rating/tags; deletion and
restoration can toggle repeatedly without creating another metadata version.
Refetching current state makes delayed or duplicate notifications harmless; never
apply their historical payload as an authoritative replica update.

## Bootstrap without losing concurrent notifications

1. Start the durable receiver and enable the webhook **before** requesting the
   full export. Keep queuing notifications while downloading.
2. Import the validated export into a fresh staging generation. Keep the previous
   generation available until staging is ready.
3. Refresh the image IDs buffered during download into staging, using the rules
   above. Keep notifications arriving during those reads queued for another read.
4. Coordinate the consumer and the generation switch locally: finish in-flight
   writes, atomically switch the active generation and consumer target, then
   process remaining notifications against the new generation. The receiver keeps
   journaling throughout this handoff.

Storage-neutral consumer sketch:

```text
onWebhook(rawBody, signature):
    verifySignature(rawBody, signature)
    persistNotification(imageID)       # before returning 2xx

refresh(imageID):                       # one worker at a time per image
    receipts = claimPendingNotifications(imageID)
    current = fetchImage(imageID)       # failures leave receipts retryable
    transaction:
        upsertOrRemove(activeGeneration, current)
        acknowledgeOnly(receipts)      # later arrivals remain pending
```

Webhooks remain best-effort: create/update enqueue failures, delivery failures,
and suspended subscriptions can leave gaps. Run a full replacement after an
outage or webhook reactivation, and periodically if eventual recovery from an
undetected gap is required. There is no incremental replay API or exactly-once
delivery guarantee.

## Server limits and verification

- `SYNC_MAX_CONCURRENT` defaults to `2` active exports per backend process.
  Excess requests receive 429 and `Retry-After: 5`.
- `SYNC_EXPORT_TIMEOUT` defaults to `5m`. Slow-client writes and database work
  are bounded by this deadline. Disconnects cancel the snapshot transaction.
- Exports read internally in batches of 1,000, keeping application memory bounded.
  Configure reverse proxies to allow streaming responses and suitable timeouts.

Backend integration tests use isolated temporary databases created through
`TEST_DATABASE_URL`. Run `go test ./...` from `backend`, frontend tests/build from
`frontend`, and `python -m unittest discover -s docs/examples -p 'test_*.py'` from the repository root for the
download example.
