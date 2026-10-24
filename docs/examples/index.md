---
title: Integration examples
---

<script setup>
import downloadUrl from './fetch_catalog.py?url'
</script>

# Integration examples

## Full catalog download

This Python example downloads a [full catalog export](/catalog-sync), checks its
manifest, record count, checksum, completion marker, and gzip integrity, then
atomically replaces a local file. It uses only the Python standard library.

<a :href="downloadUrl" download="fetch_catalog.py">Download fetch_catalog.py</a>

If you cloned the repository, run it from the repository root:

```sh
export LONGHUB_APP_KEY='long-your-app-key'
python docs/examples/fetch_catalog.py https://api.longhub.top staging-images.ndjson
```

If you downloaded the file separately, use `python fetch_catalog.py` instead.
The destination directory must already exist. The output contains image NDJSON
records without the manifest or end record; a failed download preserves any
previous output file.

For a continuously updated replica, start a durable webhook receiver before
downloading and follow the [bootstrap handoff](/catalog-sync#bootstrap-without-losing-concurrent-notifications).
The script handles download and validation; your application chooses its local
database and processes notifications by refetching current image state.

### Source

<<< ./fetch_catalog.py

### Tests

From the repository root:

```sh
python -m unittest discover -s docs/examples -p 'test_*.py' -v
```

Tests cover stream validation, truncated or corrupted downloads, and preservation
of the previous local file on failure.

## Webhook receiver

The [webhook guide](/webhook#verification) includes Python and JavaScript receiver
examples that verify the HMAC signature. For catalog synchronization, persist the
affected image ID before acknowledging a notification, then refresh it through
the image API. Handle `create`, `update`, `delete`, and `restore` events.
