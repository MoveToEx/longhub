import { zodResolver } from "@hookform/resolvers/zod";
import { Dialog as BaseDialog } from "@base-ui/react";
import { KeyRound, Plus, Send, Trash2 } from "lucide-react";
import { lazy, Suspense, useEffect, useMemo, useState } from "react";
import { Controller, useFieldArray, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import z from "zod";

import useWebhooks from "@/features/settings/hooks/use-webhooks";
import { MAX_WEBHOOK_HEADERS, MAX_WEBHOOK_HEADERS_SIZE, webhookHeaderNameError } from '@/features/settings/lib/webhook-headers';
import { Button } from "@/shared/components/ui/button";
import { Checkbox } from "@/shared/components/ui/checkbox";
import {
  Dialog,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/shared/components/ui/dialog";
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/shared/components/ui/field";
import { Input } from "@/shared/components/ui/input";
import { ResponsiveDialogCancel, ResponsiveDialogContent } from "@/shared/components/ui/responsive-dialog";
import { Spinner } from "@/shared/components/ui/spinner";
import { Switch } from "@/shared/components/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/components/ui/tooltip';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/shared/components/ui/select";
import api from "@/shared/lib/axios";
import type { Webhook } from "@/shared/lib/types";
import { formatError } from "@/shared/lib/utils";

export const WEBHOOK_EVENTS = [
  { value: 1, label: 'Creation' },
  { value: 2, label: 'Update' },
  { value: 4, label: 'Deletion / restoration' },
] as const;

const DEFAULT_BODY_TEMPLATE = '{\n  "event": {{json .Event}},\n  "data": {{json .Data}},\n  "initiator": {{json .Initiator}}\n}';
const PREVIEW_EVENTS = [
  { value: 'create', label: 'Creation' },
  { value: 'update', label: 'Update' },
  { value: 'delete', label: 'Deletion' },
  { value: 'restore', label: 'Restoration' },
];
const WebhookBodyEditor = lazy(() => import('@/features/settings/components/webhook-body-editor'));
const WebhookBodyPreview = lazy(() => import('@/features/settings/components/webhook-body-preview'));

const baseSchema = z.object({
  label: z.string().trim().min(1, 'Enter a label.'),
  endpoint: z.string().trim().refine((value) => {
    try {
      const url = new URL(value);
      return url.protocol === 'http:' || url.protocol === 'https:';
    }
    catch {
      return false;
    }
  }, 'Enter a valid HTTP or HTTPS URL.'),
  eventTypes: z.number().int().refine(value => value > 0, 'Select at least one event.'),
  secret: z.string(),
  active: z.boolean(),
  bodyTemplate: z.string(),
  headerTemplates: z.array(z.object({
    name: z.string().superRefine((value, ctx) => {
      const error = webhookHeaderNameError(value);
      if (error) ctx.addIssue({ code: 'custom', message: error });
    }),
    value: z.string(),
  })).max(MAX_WEBHOOK_HEADERS, 'At most 32 custom headers are allowed.').superRefine((headers, ctx) => {
    const names = new Set<string>();
    for (const [index, header] of headers.entries()) {
      const name = header.name.toLowerCase();
      if (names.has(name)) ctx.addIssue({ code: 'custom', path: [index, 'name'], message: 'Header names must be unique (case-insensitive).' });
      names.add(name);
    }
    if (headers.reduce((size, header) => size + new TextEncoder().encode(header.name + header.value).length, 0) > MAX_WEBHOOK_HEADERS_SIZE) {
      ctx.addIssue({ code: 'custom', message: 'Header templates exceed 64 KiB.' });
    }
  }),
});

function schemaFor(mode: 'add' | 'edit') {
  return baseSchema.refine(
    data => mode === 'edit' || data.secret.trim().length > 0,
    { path: ['secret'], message: 'Enter a signing secret.' },
  ).refine(
    data => data.bodyTemplate.trim().length > 0,
    { path: ['bodyTemplate'], message: 'Enter a body template.' },
  ).refine(
    data => new TextEncoder().encode(data.bodyTemplate).length <= 64 * 1024,
    { path: ['bodyTemplate'], message: 'Body template is too large.' },
  );
}

type WebhookForm = z.infer<typeof baseSchema>;

function WebhookFormContent({
  mode,
  webhook,
  onClose,
}: {
  mode: 'add' | 'edit',
  webhook?: Webhook,
  onClose: () => void,
}) {
  const [loading, setLoading] = useState(false);
  const [previewEvent, setPreviewEvent] = useState('update');
  const { mutate } = useWebhooks();
  const schema = useMemo(() => schemaFor(mode), [mode]);
  const form = useForm<WebhookForm>({
    resolver: zodResolver(schema),
    defaultValues: {
      label: webhook?.label ?? '',
      endpoint: webhook?.endpoint ?? '',
      eventTypes: webhook?.eventTypes ?? 0,
      secret: '',
      active: webhook?.active ?? true,
      bodyTemplate: webhook?.bodyTemplate ?? DEFAULT_BODY_TEMPLATE,
      headerTemplates: Object.entries(webhook?.headerTemplates ?? {}).map(([name, value]) => ({ name, value })),
    },
  });
  const { fields: headerFields, append: appendHeader, remove: removeHeader } = useFieldArray({ control: form.control, name: 'headerTemplates' });
  const bodyTemplate = useWatch({ control: form.control, name: 'bodyTemplate' });
  const headerTemplates = useWatch({ control: form.control, name: 'headerTemplates' });

  useEffect(() => {
    form.reset({
      label: webhook?.label ?? '',
      endpoint: webhook?.endpoint ?? '',
      eventTypes: webhook?.eventTypes ?? 0,
      secret: '',
      active: webhook?.active ?? true,
      bodyTemplate: webhook?.bodyTemplate ?? DEFAULT_BODY_TEMPLATE,
      headerTemplates: Object.entries(webhook?.headerTemplates ?? {}).map(([name, value]) => ({ name, value })),
    });
  }, [form, webhook]);

  const submit = async (values: WebhookForm) => {
    setLoading(true);

    try {
      const payload: Partial<Omit<WebhookForm, 'headerTemplates'>> & { headerTemplates: Record<string, string> } = {
        label: values.label,
        endpoint: values.endpoint,
        eventTypes: values.eventTypes,
        active: values.active,
        bodyTemplate: values.bodyTemplate,
        headerTemplates: Object.fromEntries(values.headerTemplates.map(({ name, value }) => [name, value])),
      };
      if (values.secret.trim()) payload.secret = values.secret;

      if (mode === 'add') {
        await api.post('/user/webhook', payload);
      }
      else {
        await api.patch(`/user/webhook/${webhook?.id}`, payload);
      }

      await mutate();
      onClose();
      toast.success(mode === 'add' ? 'Webhook added' : 'Webhook saved');
    }
    catch (error) {
      toast.error(formatError(error));
    }
    finally {
      setLoading(false);
    }
  };

  return (
    <ResponsiveDialogContent className='flex flex-col overflow-hidden md:max-h-[calc(100dvh-2rem)] md:w-[calc(100%-2rem)] md:max-w-6xl'>
      <DialogHeader>
        <DialogTitle className='text-xl'>
          {mode === 'add' ? 'New webhook' : 'Edit webhook'}
        </DialogTitle>
        <DialogDescription>
          Refer to our <a href="https://docs.longhub.top/webhook" target="_blank" rel="noreferrer">Docs</a> for more details
        </DialogDescription>
      </DialogHeader>

      <form onSubmit={form.handleSubmit(submit)} className='flex min-h-0 flex-col'>
        <div className='grid min-h-0 grid-cols-1 gap-6 overflow-y-auto px-1 pb-1 md:grid-cols-[minmax(0,2fr)_minmax(0,3fr)]'>
          <FieldGroup className='min-w-0 gap-5'>
            <Controller
              name='active'
              control={form.control}
              render={({ field }) => (
                <Field orientation='horizontal' className='items-center gap-4 rounded-md'>
                  <div className='flex flex-col gap-1'>
                    <FieldLabel htmlFor='form-webhook-active'>Active</FieldLabel>
                  </div>
                  <Switch
                    id='form-webhook-active'
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    aria-label='Active webhook'
                  />
                </Field>
              )} />
            <Controller
              name='label'
              control={form.control}
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel htmlFor='form-webhook-label'>Label</FieldLabel>
                  <Input {...field} id='form-webhook-label' placeholder='Production image sync' />
                  {fieldState.invalid && <FieldError errors={[fieldState.error]} />}
                </Field>
              )} />


            <Controller
              name='endpoint'
              control={form.control}
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel htmlFor='form-webhook-endpoint'>Endpoint URL</FieldLabel>
                  <Input
                    {...field}
                    id='form-webhook-endpoint'
                    type='url'
                    placeholder='https://example.com/webhooks/longhub'
                  />
                  {fieldState.invalid && <FieldError errors={[fieldState.error]} />}
                </Field>
              )} />

            <Controller
              name='eventTypes'
              control={form.control}
              render={({ field, fieldState }) => (
                <FieldSet data-invalid={fieldState.invalid}>
                  <FieldLegend variant='label'>Events</FieldLegend>
                  <div className='grid gap-2 @sm/field-group:grid-cols-3'>
                    {WEBHOOK_EVENTS.map(event => {
                      const label = event.value === 4 ? 'Deletion' : event.label;
                      const checkbox = (
                        <FieldLabel key={event.value}>
                          <Field orientation='horizontal'>
                            <Checkbox
                              aria-label={label}
                              checked={(field.value & event.value) !== 0}
                              onCheckedChange={(checked) => {
                                field.onChange(checked
                                  ? field.value | event.value
                                  : field.value & ~event.value);
                              }}
                            />
                            <div className='flex flex-col gap-1'>
                              <span>{label}</span>
                            </div>
                          </Field>
                        </FieldLabel>
                      );
                      return event.value === 4 ? (
                        <Tooltip key={event.value}>
                          <TooltipTrigger render={checkbox} />
                          <TooltipContent>Also includes restoration</TooltipContent>
                        </Tooltip>
                      ) : checkbox;
                    })}
                  </div>
                  {fieldState.invalid && <FieldError errors={[fieldState.error]} />}
                </FieldSet>
              )} />

            <Controller
              name='secret'
              control={form.control}
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel htmlFor='form-webhook-secret'>Signing secret</FieldLabel>
                  <div className='relative'>
                    <KeyRound className='text-muted-foreground absolute left-3 top-1/2 size-4 -translate-y-1/2' />
                    <Input
                      {...field}
                      id='form-webhook-secret'
                      type='password'
                      autoComplete='new-password'
                      className='pl-9'
                      placeholder={mode === 'edit' ? 'Leave blank to keep the current secret' : 'Enter a strong secret'}
                    />
                  </div>
                  <FieldDescription>
                    {mode === 'edit'
                      ? 'For security, the existing secret cannot be displayed.'
                      : 'Store this same value in the service receiving your webhook.'}
                  </FieldDescription>
                  {fieldState.invalid && <FieldError errors={[fieldState.error]} />}
                </Field>
              )} />
            <FieldSet className='gap-3'>
              <FieldLegend variant='label' className='mb-0 flex w-full items-center justify-between gap-2 py-2'>
                <span>Custom headers</span>
                <Button type='button' variant='outline' size='sm' disabled={headerFields.length >= MAX_WEBHOOK_HEADERS} onClick={() => appendHeader({ name: '', value: '' })}>
                  <Plus /> Add header
                </Button>
              </FieldLegend>
              {headerFields.length > 0 && (
                <div className='max-h-64 space-y-2 overflow-y-auto rounded-md border p-2'>
                  {headerFields.map((header, index) => (
                    <div key={header.id} className='grid grid-cols-[minmax(0,2fr)_minmax(0,3fr)_auto] items-center gap-2'>
                      <Controller
                        name={`headerTemplates.${index}.name`}
                        control={form.control}
                        render={({ field, fieldState }) => (
                          <Field data-invalid={fieldState.invalid} className='min-w-0'>
                            <Input {...field} id={`form-webhook-header-name-${index}`} aria-label='Header name' placeholder='Authorization' autoComplete='off' spellCheck={false} />
                            {fieldState.invalid && <FieldError errors={[fieldState.error]} />}
                          </Field>
                        )} />
                      <Controller
                        name={`headerTemplates.${index}.value`}
                        control={form.control}
                        render={({ field, fieldState }) => (
                          <Field data-invalid={fieldState.invalid} className='min-w-0'>
                            <Input {...field} id={`form-webhook-header-value-${index}`} aria-label='Value template' placeholder='Bearer your-token' autoComplete='off' spellCheck={false} className='font-mono text-xs' />
                            {fieldState.invalid && <FieldError errors={[fieldState.error]} />}
                          </Field>
                        )} />
                      <Button type='button' variant='ghost' size='icon' aria-label={`Remove header ${index + 1}`} onClick={() => removeHeader(index)}>
                        <Trash2 />
                      </Button>
                    </div>
                  ))}
                </div>
              )}
              <FieldError errors={[form.formState.errors.headerTemplates?.root ?? form.formState.errors.headerTemplates]} />
            </FieldSet>
          </FieldGroup>
          <FieldGroup className='min-w-0 md:border-l md:pl-6'>
            <Controller
              name='bodyTemplate'
              control={form.control}
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel>Body schema</FieldLabel>
                  <Suspense fallback={<div className='flex min-h-48 items-center justify-center'><Spinner /></div>}>
                    <WebhookBodyEditor
                      value={field.value}
                      onChange={field.onChange}
                      onBlur={field.onBlur}
                      label='Body schema'
                    />
                  </Suspense>
                  {fieldState.invalid && <FieldError errors={[fieldState.error]} />}
                </Field>
              )} />
            <Field>
              <div className='flex flex-wrap items-center justify-between gap-2'>
                <FieldLabel>Request preview</FieldLabel>
                <Select items={PREVIEW_EVENTS} value={previewEvent} onValueChange={value => { if (value) setPreviewEvent(value); }}>
                  <SelectTrigger className='w-32' aria-label='Preview event'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {PREVIEW_EVENTS.map(event => <SelectItem key={event.value} value={event.value}>{event.label}</SelectItem>)}
                  </SelectContent>
                </Select>
              </div>
              <Suspense fallback={<div className='flex min-h-32 items-center justify-center'><Spinner /></div>}>
                <WebhookBodyPreview
                  template={bodyTemplate}
                  event={previewEvent}
                  headerTemplates={headerTemplates}
                />
              </Suspense>
            </Field>
          </FieldGroup>
        </div>

        <DialogFooter className='mt-6'>
          <Button type='submit' disabled={loading}>
            {loading ? <Spinner /> : <Send />}
            {mode === 'add' ? 'Add webhook' : 'Save changes'}
          </Button>
          <ResponsiveDialogCancel disabled={loading} />
        </DialogFooter>
      </form>
    </ResponsiveDialogContent>
  );
}

export function AddWebhookDialog({ handle }: { handle: BaseDialog.Handle<void> }) {
  return (
    <Dialog handle={handle}>
      <WebhookFormContent mode='add' onClose={() => handle.close()} />
    </Dialog>
  );
}

export function EditWebhookDialog({ handle }: { handle: BaseDialog.Handle<Webhook> }) {
  return (
    <Dialog<Webhook> handle={handle}>
      {({ payload }) => (
        <WebhookFormContent mode='edit' webhook={payload} onClose={() => handle.close()} />
      )}
    </Dialog>
  );
}
