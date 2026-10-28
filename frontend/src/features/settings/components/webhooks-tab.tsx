import { Dialog as BaseDialog } from "@base-ui/react";
import {
  Activity,
  CircleAlert,
  CircleCheck,
  Clock3,
  Edit,
  EllipsisVertical,
  Plus,
  Trash,
  Webhook as WebhookIcon,
} from "lucide-react";

import DeleteWebhookDialog from "@/features/settings/components/webhook-delete-dialog";
import {
  AddWebhookDialog,
  EditWebhookDialog,
  WEBHOOK_EVENTS,
} from "@/features/settings/components/webhook-dialog";
import useWebhooks from "@/features/settings/hooks/use-webhooks";
import { Alert, AlertDescription, AlertTitle } from "@/shared/components/ui/alert";
import { Button } from "@/shared/components/ui/button";
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/shared/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/shared/components/ui/dropdown-menu";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/shared/components/ui/empty";
import { Spinner } from "@/shared/components/ui/spinner";
import type { Webhook } from "@/shared/lib/types";

const MAX_WEBHOOKS = 10;
const addHandle = BaseDialog.createHandle<void>();
const editHandle = BaseDialog.createHandle<Webhook>();
const deleteHandle = BaseDialog.createHandle<Webhook>();

function eventLabels(mask: number) {
  return WEBHOOK_EVENTS.filter(event => (mask & event.value) !== 0);
}

function statusDetails(webhook: Webhook) {
  if (!webhook.active && webhook.failureCount >= 3) {
    return { label: 'Paused after failures', className: 'text-destructive', icon: CircleAlert };
  }
  if (!webhook.active) {
    return { label: 'Inactive', className: 'text-muted-foreground', icon: CircleAlert };
  }
  if (webhook.lastResponseStatus === null) {
    return { label: 'Waiting for first delivery', className: 'text-muted-foreground', icon: Clock3 };
  }
  if (webhook.lastResponseStatus >= 200 && webhook.lastResponseStatus < 300) {
    return { label: `Healthy · ${webhook.lastResponseStatus}`, className: 'text-emerald-600 dark:text-emerald-400', icon: CircleCheck };
  }
  return { label: `Last response · ${webhook.lastResponseStatus}`, className: 'text-amber-600 dark:text-amber-400', icon: CircleAlert };
}

function formatLastDelivery(value: string | null) {
  if (!value || value.startsWith('0001-')) return 'Never delivered';
  return `Last delivered ${new Date(value).toLocaleString()}`;
}

function WebhookCard({ webhook }: { webhook: Webhook }) {
  const status = statusDetails(webhook);
  const StatusIcon = status.icon;

  return (
    <Card size='sm'>
      <CardHeader>
        <CardTitle className='flex items-center gap-2'>
          <WebhookIcon className='size-4' />
          {webhook.label}
        </CardTitle>
        <CardDescription className='break-all font-mono text-xs'>
          {webhook.endpoint}
        </CardDescription>
        <CardAction>
          <DropdownMenu>
            <DropdownMenuTrigger render={<Button variant='outline' size='icon-sm' />}>
              <EllipsisVertical />
              <span className='sr-only'>Webhook actions</span>
            </DropdownMenuTrigger>
            <DropdownMenuContent align='end'>
              <DropdownMenuItem onClick={() => editHandle.openWithPayload(webhook)}>
                <Edit /> Edit
              </DropdownMenuItem>
              <DropdownMenuItem variant='destructive' onClick={() => deleteHandle.openWithPayload(webhook)}>
                <Trash /> Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </CardAction>
      </CardHeader>
      <CardContent className='flex flex-col gap-3'>
        <div className='flex flex-wrap gap-2'>
          {eventLabels(webhook.eventTypes).map(event => (
            <span key={event.value} className='bg-muted rounded-full px-2.5 py-1 text-xs'>
              {event.label}
            </span>
          ))}
        </div>
        <div className='text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1 text-xs'>
          <span className={`flex items-center gap-1.5 ${status.className}`}>
            <StatusIcon className='size-3.5' />
            {status.label}
          </span>
          <span className='flex items-center gap-1.5'>
            <Activity className='size-3.5' />
            {formatLastDelivery(webhook.lastActivatedAt)}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

export default function WebhooksTab() {
  const { data, error, isLoading } = useWebhooks();
  const atLimit = (data?.length ?? 0) >= MAX_WEBHOOKS;

  return (
    <div className='w-full flex flex-col items-center'>
      <div className='w-full md:w-3/4 flex flex-col gap-5'>
        <AddWebhookDialog handle={addHandle} />
        <EditWebhookDialog handle={editHandle} />
        <DeleteWebhookDialog handle={deleteHandle} />

        <div>
          <h2 className='text-lg font-medium'>Webhooks</h2>
          <p className='text-muted-foreground mt-1 text-sm'>
            Send image lifecycle events to your application in real time.
          </p>
        </div>

        {error && (
          <Alert variant='destructive'>
            <CircleAlert />
            <AlertTitle>Could not load webhooks</AlertTitle>
            <AlertDescription>Refresh the page and try again.</AlertDescription>
          </Alert>
        )}

        {isLoading && (
          <div className='text-muted-foreground flex h-20 items-center justify-center gap-2'>
            <Spinner /> Loading webhooks...
          </div>
        )}

        {!isLoading && data?.length === 0 && (
          <Empty className='min-h-48 border'>
            <EmptyHeader>
              <EmptyMedia variant='icon'><WebhookIcon /></EmptyMedia>
              <EmptyTitle>No webhooks yet</EmptyTitle>
              <EmptyDescription>
                Add an endpoint to receive image created, updated, and deleted events.
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        )}

        <div className='flex flex-col gap-3'>
          {data?.map(webhook => <WebhookCard key={webhook.id} webhook={webhook} />)}
        </div>

        <div className='flex items-center justify-between gap-3'>
          <span className='text-muted-foreground text-xs'>
            {data?.length ?? 0} of {MAX_WEBHOOKS} webhooks
          </span>
          <BaseDialog.Trigger
            handle={addHandle}
            disabled={atLimit}
            render={<Button disabled={atLimit} />}>
            <Plus /> Add webhook
          </BaseDialog.Trigger>
        </div>
      </div>
    </div>
  );
}
