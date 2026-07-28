import { zodResolver } from "@hookform/resolvers/zod";
import { Dialog as BaseDialog } from "@base-ui/react";
import { KeyRound, Send } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { toast } from "sonner";
import z from "zod";

import useWebhooks from "@/features/settings/hooks/use-webhooks";
import { Button } from "@/shared/components/ui/button";
import { Checkbox } from "@/shared/components/ui/checkbox";
import {
  Dialog,
  DialogClose,
  DialogContent,
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
import { Spinner } from "@/shared/components/ui/spinner";
import { Switch } from "@/shared/components/ui/switch";
import api from "@/shared/lib/axios";
import type { Webhook } from "@/shared/lib/types";
import { formatError } from "@/shared/lib/utils";

export const WEBHOOK_EVENTS = [
  { value: 1, label: 'Creation', description: 'When an image is uploaded.' },
  { value: 2, label: 'Update', description: 'When an image have its metadata changed.' },
  { value: 4, label: 'Deletion', description: 'When an image is deleted.' },
] as const;

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
});

function schemaFor(mode: 'add' | 'edit') {
  return baseSchema.refine(
    data => mode === 'edit' || data.secret.trim().length > 0,
    { path: ['secret'], message: 'Enter a signing secret.' },
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
    },
  });

  useEffect(() => {
    form.reset({
      label: webhook?.label ?? '',
      endpoint: webhook?.endpoint ?? '',
      eventTypes: webhook?.eventTypes ?? 0,
      secret: '',
      active: webhook?.active ?? true,
    });
  }, [form, webhook]);

  const submit = async (values: WebhookForm) => {
    setLoading(true);

    try {
      const payload: Partial<WebhookForm> = {
        label: values.label,
        endpoint: values.endpoint,
        eventTypes: values.eventTypes,
        active: values.active,
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
    <DialogContent className='sm:max-w-xl'>
      <DialogHeader>
        <DialogTitle className='text-xl'>
          {mode === 'add' ? 'New webhook' : 'Edit webhook'}
        </DialogTitle>
        <DialogDescription>
          LONG Hub signs each request body with HMAC-SHA256 and sends the signature in the X-Signature header.
        </DialogDescription>
      </DialogHeader>

      <form onSubmit={form.handleSubmit(submit)}>
        <FieldGroup>
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
                <div className='grid gap-2 sm:grid-cols-3'>
                  {WEBHOOK_EVENTS.map(event => (
                    <FieldLabel key={event.value}>
                      <Field orientation='horizontal'>
                        <Checkbox
                          checked={(field.value & event.value) !== 0}
                          onCheckedChange={(checked) => {
                            field.onChange(checked
                              ? field.value | event.value
                              : field.value & ~event.value);
                          }}
                        />
                        <div className='flex flex-col gap-1'>
                          <span>{event.label}</span>
                          <FieldDescription className='font-xs'>{event.description}</FieldDescription>
                        </div>
                      </Field>
                    </FieldLabel>
                  ))}
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
        </FieldGroup>

        <DialogFooter className='mt-6'>
          <Button type='submit' disabled={loading}>
            {loading ? <Spinner /> : <Send />}
            {mode === 'add' ? 'Add webhook' : 'Save changes'}
          </Button>
          <DialogClose disabled={loading} render={<Button variant='outline' />}>
            Cancel
          </DialogClose>
        </DialogFooter>
      </form>
    </DialogContent>
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
