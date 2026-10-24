<OAIntroduction />

<script setup>
import jsonSpecUrl from '../spec/openapi.json?url'
import yamlSpecUrl from '../spec/openapi.yaml?url'
</script>

Download the specification:
<a :href="jsonSpecUrl" download="openapi.json">OpenAPI JSON</a> ·
<a :href="yamlSpecUrl" download="openapi.yaml">OpenAPI YAML</a>.

## Authentication

Most read-only operations do not require authorization or authentication. You can directly invoke these endpoints. To use writing or account-related operations, see below.  

The [full catalog export](/api/exportFullCatalog) requires authentication even
though it is read-only. It accepts any valid app key or session and requires no
write permission. See [catalog sync](/catalog-sync) for the streaming protocol and
[examples](/examples/) for a download client.

Two types of authorization are supported, and either of them can be used to authenticate both on web and via API:

### Appkey

:::tip
Authenticating via appkey should be preferred over sessions.
:::

You need an appkey to authenticate this way. Go to _Settings_ > _Appkeys_ and create one.  
Keep your appkey at a secure place, you will not see it after the creation dialog closes.  

To authenticate, use the `Authorization` header with `Key` type:

```http
POST https://api.longhub.top/image/sign
Authorization: Key <YOUR_APP_KEY>
...
```

### Session

Use the session token returned by login with the `Session` authorization type:

```http
GET https://api.longhub.top/sync/full
Authorization: Session <SESSION_TOKEN>
```
