"""Download and validate a full export using only the Python standard library.

Set LONGHUB_APP_KEY, then run:
    python examples/fetch_catalog.py https://api.example.com catalog.ndjson

The output contains image records only. A webhook receiver must journal affected
image IDs before this download starts; see docs/catalog-sync.md for the handoff.
"""

import argparse
import gzip
import hashlib
import json
import os
from pathlib import Path
import tempfile
from urllib.request import Request, urlopen


def copy_export(source, output):
    """Copy image lines to staging, accepting only a complete, verified export."""
    manifest = None
    complete = False
    count = 0
    digest = hashlib.sha256()
    for line in source:
        if complete or not line.endswith(b"\n"):
            raise ValueError("Unexpected data after end or incomplete NDJSON line")
        record = json.loads(line)
        if not isinstance(record, dict):
            raise ValueError("Expected an NDJSON object")
        if manifest is None:
            if (record.get("type") != "manifest"
                    or record.get("schemaVersion") != 1
                    or not record.get("exportId")):
                raise ValueError("Unsupported or missing export manifest")
            manifest = record
        elif record.get("type") == "image":
            if not isinstance(record.get("data"), dict):
                raise ValueError("Missing image data")
            digest.update(line)
            output.write(line)
            count += 1
        elif record.get("type") == "end":
            if (record.get("exportId") != manifest["exportId"]
                    or record.get("recordCount") != count
                    or record.get("sha256") != digest.hexdigest()):
                raise ValueError("Export count, checksum, or identifier mismatch")
            complete = True
        else:
            raise ValueError("Unexpected export record")
    if not complete:
        raise ValueError("Export interrupted: missing end record")
    return count


def fetch_catalog(base_url, destination, app_key):
    """Publish the downloaded file only after validating the entire response."""
    destination = Path(destination)
    request = Request(base_url.rstrip("/") + "/sync/full", headers={
        "Authorization": "Key " + app_key,
        "Accept-Encoding": "gzip",
    })
    staging = None
    try:
        with urlopen(request, timeout=330) as response:
            encoding = response.headers.get("Content-Encoding", "").lower()
            if encoding not in ("", "gzip"):
                raise ValueError("Unsupported response encoding")
            source = gzip.GzipFile(fileobj=response) if encoding == "gzip" else response
            with source, tempfile.NamedTemporaryFile(
                dir=destination.parent, prefix=destination.name + ".", suffix=".tmp", delete=False,
            ) as output:
                staging = Path(output.name)
                count = copy_export(source, output)
                output.flush()
                os.fsync(output.fileno())
        os.replace(staging, destination)
        return count
    finally:
        if staging is not None:
            staging.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("base_url")
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    key = os.environ.get("LONGHUB_APP_KEY")
    if not key:
        parser.error("Set LONGHUB_APP_KEY to an app key before downloading")
    count = fetch_catalog(args.base_url, args.output, key)
    print(f"Validated {count} images; saved to {args.output}")


if __name__ == "__main__":
    main()
