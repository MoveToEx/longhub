import { useEffect, useState } from 'react';

import WebhookBodyEditor from '@/features/settings/components/webhook-body-editor';
import { renderWebhookBodyPreview } from '@/features/settings/lib/webhook-body-preview';
import { Spinner } from '@/shared/components/ui/spinner';
import { formatError } from '@/shared/lib/utils';

type PreviewState = {
  template: string,
  event: string,
  body?: string,
  error?: string,
};

export default function WebhookBodyPreview({ template, event }: { template: string, event: string }) {
  const [preview, setPreview] = useState<PreviewState | null>(null);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      try {
        setPreview({ template, event, body: renderWebhookBodyPreview(template, event) });
      }
      catch (error) {
        setPreview(previous => ({ template, event, body: previous?.body, error: formatError(error) }));
      }
    }, 350);
    return () => {
      window.clearTimeout(timer);
    };
  }, [template, event]);

  const pending = !preview || preview.template !== template || preview.event !== event;
  const error = pending ? undefined : preview.error;
  const stale = pending || Boolean(error);

  return (
    <div className='min-w-0 space-y-2' aria-busy={pending}>
      {preview?.body !== undefined ? (
        <div className={stale ? 'opacity-50 grayscale' : undefined}>
          <WebhookBodyEditor value={preview.body} readOnly jsonSyntax label='Request preview' />
          {pending && <span className='sr-only' role='status'>Rendering preview...</span>}
        </div>
      ) : pending ? (
        <div className='text-muted-foreground flex min-h-32 items-center justify-center gap-2' role='status'><Spinner /> Rendering preview...</div>
      ) : null}
      {error && <p className='text-destructive break-words rounded-md border p-3 text-sm' role='alert'>{error}</p>}
    </div>
  );
}
