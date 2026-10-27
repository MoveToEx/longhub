# Replicating the image catalog

Use a full export to bootstrap or replace a local catalog, then use webhooks as
notifications to refresh individual images.

## Check the catalog version

Send [`GET /sync/version`](/api/getSyncVersion) with either
`Authorization: Key <app-key>` or `Authorization: Session <session-token>`,
as with `/sync/full`:

```sh
curl --fail \
  -H "Authorization: Key $LONGHUB_APP_KEY" \
  https://api.example.com/sync/version
```

The JSON response uses the standard envelope:

```json
{"error":null,"data":{"timestamp":"2026-10-27T08:30:00.123456Z"}}
```

`timestamp` is the latest recorded catalog change as a UTC RFC 3339 timestamp.
It includes image uploads, metadata edits (including tags and ratings), deletions,
and restores. It is `null` when no catalog change has been recorded. Repeated
checks return the same value until a later change is recorded; reads, exports,
and background webhook or indexing work do not advance it. Responses use
`Cache-Control: no-store`.

Use this value to decide whether to refresh a local catalog. It is a change hint,
not an incremental replay cursor or a guarantee that an export includes changes
committed concurrently with it.

## Download a full export

Send `GET /sync/full` with either `Authorization: Key <app-key>` or
`Authorization: Session <session-token>`.

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
   image ID. `data` has the same shape as the `data` returned by [`GET /image/:id`](/api/getImage).
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

The standard-library [Python example](#examples)
validates the stream and atomically replaces a local file only after success:

```sh
export LONGHUB_APP_KEY='long-your-app-key'
python docs/examples/fetch_catalog.py https://api.example.com staging-images.ndjson
```

Its output contains only image record lines. This example handles download and
validation; integrate the webhook handoff below before publishing an application's
new local catalog.

## Bootstrapping

1. Start the durable receiver and enable the webhook **before** requesting the
   full export. Keep queuing notifications while downloading.
2. Import the validated export into a fresh staging generation. Keep the previous
   generation available until staging is ready.
3. Refresh the image IDs buffered during download into staging through
   `GET /image/:id`. Keep notifications arriving during those reads queued for another read.
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


## Examples

<<< ./examples/fetch_catalog.py

## Restrictions

- At most 2 concurrent sync connections are allowed. Excess requests receive 429 and `Retry-After: 5`.
- A sync connection expires after 5 minutes.
