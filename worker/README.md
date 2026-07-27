# Webhook gateway

The Worker accepts authenticated webhook invocation envelopes and forwards them
to the requested HTTP(S) URL.

Configure `SERVER_PUBLIC_KEY` with the server's Ed25519 public key. The value may
be a base64-encoded SPKI DER key, a PEM public key, or a base64-encoded 32-byte
raw public key. For example, it can be stored as a Worker secret:

```sh
npx wrangler secret put SERVER_PUBLIC_KEY
```

Send a `POST` request whose raw JSON body has this shape:

```json
{
  "url": "https://client.example/webhook",
  "body": "{\"imageID\":42}",
  "clientSignature": "0123456789abcdef"
}
```

Sign the exact UTF-8 bytes of that JSON body with the server's Ed25519 private
key and put the base64 (or hex) signature in `X-Server-Signature`. The Worker
forwards `body` unchanged as an `application/json` POST request and sends
`clientSignature` in the `X-Signature` header.
