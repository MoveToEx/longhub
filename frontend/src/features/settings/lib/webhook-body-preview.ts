import { createEngine } from '@promptctl/go-template-js';
import { MAX_WEBHOOK_HEADERS, MAX_WEBHOOK_HEADERS_SIZE, webhookHeaderNameError, type WebhookHeaderTemplate } from './webhook-headers.ts';

const MAX_TEMPLATE_SIZE = 64 * 1024;
const MAX_PAYLOAD_SIZE = 256 * 1024;
const encoder = new TextEncoder();

const engine = createEngine<string>({
  fromString: value => value,
  isT: (value): value is string => typeof value === 'string',
  missingKey: 'error',
  funcs: {
    json: {
      fn: (value: unknown) => JSON.stringify(value ?? null),
      argTypes: ['value'],
      arity: { kind: 'exact' },
    },
  },
});

// Go field names and JSON payload keys use different casing.
function templateValue<T extends object>(fields: T, jsonValue: unknown) {
  return { ...fields, toJSON: () => jsonValue };
}

function previewData(event: string) {
  return {
    imageID: 42,
    text: 'Updated image caption',
    rating: 'none',
    tags: ['landscape', 'featured'],
    imageURL: 'https://example.com/images/42.jpg',
    createdAt: '2026-01-01T12:00:00Z',
    uploader: { id: 7, username: 'image-owner' },
    ...(event === 'update' && {
      changes: {
        text: { old: 'Original image caption', new: 'Updated image caption' },
        rating: { old: 'moderate', new: 'none' },
        tags: { old: ['landscape'], new: ['featured', 'landscape'] },
      },
    }),
  };
}

function renderTemplatePreview(template: string, event: string, kind: 'Body' | 'Header'): string {
  if (!['create', 'update', 'delete'].includes(event)) throw new Error('Invalid webhook event.');
  if (encoder.encode(template).length > MAX_TEMPLATE_SIZE) throw new Error(`${kind} template exceeds 64 KiB.`);

  const data = previewData(event);
  const initiator = event === 'create' ? data.uploader : { id: 9, username: 'image-editor' };
  if (kind === 'Body' && !template.trim()) return JSON.stringify({ event, data, initiator }, null, 2);

  const changes = data.changes
    ? Object.fromEntries(Object.entries(data.changes).map(([key, change]) => [
        key, templateValue({ Old: change.old, New: change.new }, change),
      ]))
    : null;
  const fields = {
    ImageID: data.imageID,
    Text: data.text,
    Rating: data.rating,
    Tags: data.tags,
    ImageURL: data.imageURL,
    CreatedAt: data.createdAt,
    Changes: changes,
    Uploader: templateValue({ ID: data.uploader.id, Username: data.uploader.username }, data.uploader),
  };
  const scope = templateValue({
    Event: event,
    Data: templateValue(fields, data),
    Initiator: templateValue({ ID: initiator.id, Username: initiator.username }, initiator),
    ...fields,
  }, { Event: event, ...data, Initiator: initiator });

  let body: string;
  try {
    body = engine.parse(template).evaluate(scope).join('');
  }
  catch (error) {
    throw new Error(`Invalid ${kind.toLowerCase()} template: ${error instanceof Error ? error.message : String(error)}`);
  }
  if (kind === 'Header') {
    if (encoder.encode(body).length > 8 * 1024) throw new Error('Rendered header value exceeds 8 KiB.');
    for (const c of body) {
      if (c !== '\t' && (c < ' ' || c > '~')) throw new Error('Header values must contain only ASCII text without control characters.');
    }
    return body.replace(/^[ \t]+|[ \t]+$/g, '');
  }
  if (encoder.encode(body).length > MAX_PAYLOAD_SIZE) throw new Error('Rendered body exceeds 256 KiB.');
  try {
    return JSON.stringify(JSON.parse(body), null, 2);
  }
  catch {
    throw new Error('Body template must render valid JSON.');
  }
}

export function renderWebhookBodyPreview(template: string, event: string): string {
  return renderTemplatePreview(template, event, 'Body');
}

export function renderWebhookHeadersPreview(headers: WebhookHeaderTemplate[], event: string): string {
  if (headers.length > MAX_WEBHOOK_HEADERS) throw new Error('At most 32 custom headers are allowed.');
  const names = new Set<string>();
  let sourceSize = 0;
  let renderedSize = 0;
  return headers.map(({ name, value }) => {
    const error = webhookHeaderNameError(name);
    if (error) throw new Error(`${name || 'Header'}: ${error}`);
    const normalized = name.toLowerCase();
    if (names.has(normalized)) throw new Error(`Duplicate header: ${name}.`);
    names.add(normalized);
    sourceSize += encoder.encode(name + value).length;
    if (sourceSize > MAX_WEBHOOK_HEADERS_SIZE) throw new Error('Header templates exceed 64 KiB.');
    const rendered = renderTemplatePreview(value, event, 'Header');
    renderedSize += name.length + rendered.length;
    if (renderedSize > MAX_WEBHOOK_HEADERS_SIZE) throw new Error('Rendered headers exceed 64 KiB.');
    return `${name}: ${rendered}`;
  }).join('\n');
}
