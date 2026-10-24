---
title: Webhook
---

# Webhook

Webhooks allow an external server to get notified when certain events happen.

## Prerequisites

- A server reachable from the Internet (through public IP or port forwarding)
- Ability to write (or whip your coding agent to write) server-side programs that work with our schema

## Events

| Subscription | Bit | Event names |
| --- | --- | --- |
| Creation | 1 | `create` |
| Update | 2 | `update` |
| Deletion / restoration | 4 | `delete`, `restore` |

Set `eventTypes` to `7` to receive all events. Restoring an image belongs to the
existing deletion subscription and emits `restore`; it does not create a metadata
version. Receivers and custom templates should handle both lifecycle event names.

For a local replica, treat every notification as a request to refresh the image
through `GET /image/:id`. The current response determines whether to upsert or
remove it. Delayed events describe past actions and must not overwrite newer
local state directly. See [catalog sync](/catalog-sync) and the
[Python example](/catalog-sync#examples) for bootstrap and recovery.

## Schema

All webhooks are sent as POST requests with a customizable JSON body.  
The schema specification uses [`text/template`](https://pkg.go.dev/text/template) package from Golang.  

The following fields are exposed at root level:

```ts
{
  Event: 'create' | 'update' | 'delete' | 'restore',
  Initiator: {    // user whose action triggered this webhook
    ID: number,
    Username: string,
  },
  Data: {
    ImageID: number,
    Text: string,
    Rating: 'none' | 'moderate' | 'violent',
    Tags: string[],
    ImageURL: string,
    CreatedAt: string, // ISO 8601
    Uploader: {   // image owner
      ID: number,
      Username: string,
    },
    Changes: {
      text?: {
        Old: string,
        New: string,
      },
      tags?: {
        Old: string[],
        New: string[]
      },
      rating?: {
        Old: 'none' | 'moderate' | 'violent',
        New: 'none' | 'moderate' | 'violent',
      }
    }
  }
}
```

Each image field also has a root alias, such as `.Text` or `.Changes`.  

`.Data.Uploader` identifies the image owner; `.Initiator` identifies the user whose action triggered the event.  

Use `json` to encode values, including quotes, newlines, arrays, and timestamps.  
Do not surround <code v-pre>{{json}}</code> expressions with additional quotes:

```
{
  "event": {{json .Event}},
  "data": {{json .Data}},
  "initiator": {{json .Initiator}}
}
```


For update events, `data.changes` contains only changed metadata fields. Each entry
has `old` and `new` values. Text and rating are compared directly; tags are compared
as unordered sets of database tags. An update without metadata changes has an
empty changes object. Creation, deletion, and restoration payloads omit `changes`.
`CreatedAt` is the image creation time, not the notification time. Deletion and
restoration payloads also include `deletionEventId` for deduplication; it is not
a revision or an ordering cursor.


```json
{
  "changes": {
    "text": { "old": "Original caption", "new": "Updated caption" },
    "tags": { "old": ["landscape"], "new": [] }
  }
}
```

Guard optional changes when referencing individual entries:

```
{
  "event": {{json .Event}},
  "previousText": {{with .Changes}}{{with .text}}{{json .Old}}{{else}}null{{end}}{{else}}null{{end}}
}
```

An example that complies with Discord Webhook:

```
{
  {{if eq .Event "create"}}
    "content": "{{.Initiator.Username}} created image #{{.Data.ImageID}}",
  {{else if eq .Event "update"}}
    "embeds": [
      {
        "type": "rich",
        "title": "{{.Initiator.Username}} updated image #{{.Data.ImageID}}",
        "url": "https://longhub.top/image/{{.Data.ImageID}}",
        "thumbnail": {
          "url": "{{.Data.ImageURL}}"
        }
      }
    ],
  {{else if eq .Event "delete"}}
    "content": "{{.Initiator.Username}} deleted image #{{.Data.ImageID}}",
  {{else if eq .Event "restore"}}
    "content": "{{.Initiator.Username}} restored image #{{.Data.ImageID}}",
  {{end}}
  "username": "LONG Hub Webhook",
  "avatar_url": "https://longhub.top/long.jpg"
}
```

The invocation request is as follows:

```http
POST https://example.com/webhook/endpoint
Content-Type: application/json
X-Signature: 0123456890abcdef

{
    ...
}
```

## Custom request headers

Add multiple header name/value pairs under **Custom headers** when creating or
editing a webhook. Names are fixed HTTP header names; values are Go text
templates with the same fields and functions as the body template. Values render
as plain text and do not need JSON quotes.

For example, the create and edit APIs accept:

```json
{
  "headerTemplates": {
    "Authorization": "Bearer your-token",
    "X-Longhub-Event": "{{.Event}}",
    "X-Image-ID": "{{.Data.ImageID}}"
  }
}
```

For image 42's restoration event, these produce `X-Longhub-Event: restore` and
`X-Image-ID: 42`. The request preview shows rendered header values for the selected
sample event. Header values are rendered from the event snapshot at delivery;
invalid output counts as a failed invocation.

`GET /user/webhook` returns the saved `headerTemplates`. In a PATCH request,
omitting this field preserves existing headers; an object replaces all pairs,
and `{}` or `null` clears them. Existing webhooks have no custom headers.

Names must be unique regardless of case. Signing headers (`X-Signature` and
`X-Server-Signature`), `Host`, `Content-Length`, `Connection`, `Keep-Alive`, `TE`,
`Trailer`, `Transfer-Encoding`, `Upgrade`, and names starting with `Proxy-` are
reserved. `Content-Type` defaults to `application/json` and can be customized.
Rendered values must contain ASCII text or tabs, without newlines or other
control characters. Each webhook supports up to 32 custom headers, 64 KiB of
combined names and template sources, 8 KiB per rendered value, and 64 KiB of
combined rendered names and values.

## Verification

In order to prove identity and integrity, webhooks are signed with the secret registered at creation. The signature is a hexadecimal string generated by signing the request body (in raw bytes encoded with UTF-8) with the webhook secret (also encoded with UTF-8) using HMAC-SHA256, and will be sent in the `X-Signature` HTTP header.  

To verify the signature, use whatever cryptography library you like. For example:

::: code-group
```py
import hashlib
import hmac
import json
import os

from flask import Flask, request, jsonify

app = Flask()
WEBHOOK_SECRET = os.environ["WEBHOOK_SECRET"].encode("utf-8")


@app.post("/hooks/longhub")
def receive_webhook():
    raw_body = request.get_data(cache=False)

    received_signature = request.headers.get("X-Signature", "")
    expected_signature = hmac.new(
        WEBHOOK_SECRET,
        raw_body,
        hashlib.sha256,
    ).hexdigest()

    if not hmac.compare_digest(received_signature, expected_signature):
        return jsonify(error="invalid signature"), 401

    payload = json.loads(raw_body)
    print("Verified webhook:", payload)

    return "", 204
```
```javascript
const express = require("express");
const crypto = require("crypto");

const app = express();

const WEBHOOK_SECRET = Buffer.from(process.env.WEBHOOK_SECRET, "utf8");

app.post("/hooks/longhub",
  express.raw({ type: "*/*" }),
  (req, res) => {
    const rawBody = req.body;

    const receivedSignature = req.get("X-Signature") || "";

    const expectedSignature = crypto
      .createHmac("sha256", WEBHOOK_SECRET)
      .update(rawBody)
      .digest("hex");

    const received = Buffer.from(receivedSignature, "utf8");
    const expected = Buffer.from(expectedSignature, "utf8");

    const valid =
      received.length === expected.length &&
      crypto.timingSafeEqual(received, expected);

    if (!valid) {
      return res.status(401).json({ error: "invalid signature" });
    }

    const payload = JSON.parse(rawBody.toString("utf8"));

    console.log("Verified webhook:", payload);

    return res.sendStatus(204);
  }
);

const port = process.env.PORT || 3000;

app.listen(port, () => {
  console.log(`Server listening on port ${port}`);
});
```
:::

## Restrictions

To protect our server from being affected by malicious invocations, we have applied the following restrictions to webhooks:  
- A user can have up to 10 webhooks.
- Requests pointing at private IP ranges are forbidden.
- Webhooks are invoked via a remote Worker. You may see an IP from Cloudflare.
- Webhooks do not follow redirects, and redirect responses (HTTP 3xx) are considered failures.
- Webhooks that fail (HTTP 3xx, 4xx, 5xx) for 3 invocations in a row will be suspended until manual re-activation.
- Webhook invocations have a timeout of 30s.
