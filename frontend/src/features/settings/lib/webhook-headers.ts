export type WebhookHeaderTemplate = { name: string, value: string };

export const MAX_WEBHOOK_HEADERS = 32;
export const MAX_WEBHOOK_HEADERS_SIZE = 64 * 1024;

const RESERVED_HEADERS = new Set([
  'x-signature', 'x-server-signature', 'host', 'content-length', 'connection',
  'keep-alive', 'te', 'trailer', 'transfer-encoding', 'upgrade',
]);

export function webhookHeaderNameError(name: string): string | undefined {
  if (!/^[!#$%&'*+.^_`|~0-9A-Za-z-]+$/.test(name)) return 'Enter a valid header name.';
  const normalized = name.toLowerCase();
  if (RESERVED_HEADERS.has(normalized) || normalized.startsWith('proxy-')) return 'This header is reserved.';
}
