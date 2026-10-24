import gzip
import hashlib
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from fetch_catalog import copy_export, fetch_catalog


def line(value):
    return json.dumps(value).encode() + b"\n"


def export():
    image = line({"type": "image", "data": {"id": 42, "text": "中文\ncaption"}})
    return (line({"type": "manifest", "schemaVersion": 1, "exportId": "test"})
            + image
            + line({"type": "end", "exportId": "test", "recordCount": 1,
                    "sha256": hashlib.sha256(image).hexdigest()}))


class ExportTest(unittest.TestCase):
    def test_validates_exact_record_bytes(self):
        output = io.BytesIO()
        self.assertEqual(copy_export(io.BytesIO(export()), output), 1)
        self.assertEqual(json.loads(output.getvalue())["data"]["id"], 42)

    def test_rejects_truncation_corruption_and_extra_records(self):
        valid = export()
        for invalid in (valid.rsplit(b"\n", 2)[0] + b"\n",
                        valid.replace(b'"recordCount": 1', b'"recordCount": 2'),
                        valid.replace(b'"id": 42', b'"id": 43'),
                        valid + b"{}\n"):
            with self.subTest(invalid=invalid):
                with self.assertRaises(ValueError):
                    copy_export(io.BytesIO(invalid), io.BytesIO())

    def test_failed_download_keeps_existing_replica(self):
        with tempfile.TemporaryDirectory() as directory:
            destination = Path(directory) / "catalog.ndjson"
            destination.write_bytes(b"previous catalog")
            response = io.BytesIO(gzip.compress(export()[:-10]))
            response.headers = {"Content-Encoding": "gzip"}
            with patch("fetch_catalog.urlopen", return_value=response):
                with self.assertRaises(ValueError):
                    fetch_catalog("https://example.com", destination, "test-key")
            self.assertEqual(destination.read_bytes(), b"previous catalog")
            self.assertEqual(list(Path(directory).iterdir()), [destination])


if __name__ == "__main__":
    unittest.main()
