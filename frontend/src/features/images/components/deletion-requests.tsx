import { useState } from "react";
import { Link } from "react-router";
import { isAxiosError } from "axios";
import useSWR, { useSWRConfig } from "swr";
import { ChevronDownIcon, Clock3, CircleCheck, CircleX, RotateCcw } from "lucide-react";
import { toast } from "sonner";
import useAuth from "@/features/auth/hooks/use-auth";
import api from "@/shared/lib/axios";
import { formatError } from "@/shared/lib/utils";
import { Button } from "@/shared/components/ui/button";
import { Card, CardContent } from "@/shared/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/shared/components/ui/collapsible";
import {
  Dialog,
  DialogTrigger,
  DialogTitle,
  DialogHeader,
  DialogFooter,
} from "@/shared/components/ui/dialog";
import {
  ResponsiveDialogContent,
  ResponsiveDialogCancel,
} from "@/shared/components/ui/responsive-dialog";
import { Field, FieldLabel } from "@/shared/components/ui/field";
import { Textarea } from "@/shared/components/ui/textarea";
import { Pagination } from "@/shared/components/ui/pagination";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/shared/components/ui/select";
import { invalidateDeletionCaches } from "../lib/deletion-cache";

type Request = {
  id: number;
  reason: string;
  status: "active" | "approved" | "rejected" | "revoked";
  requester: { id: number; username: string };
  createdAt: string;
  processedAt: string | null;
};
const statusIcons = {
  active: Clock3,
  approved: CircleCheck,
  rejected: CircleX,
  revoked: RotateCcw,
};

function RequestStatusBadge({ status }: { status: Request["status"] }) {
  const Icon = statusIcons[status];
  return (
    <span className="inline-flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 capitalize">
      <Icon aria-hidden="true" className="size-3.5" />
      {status}
    </span>
  );
}

const templates = {
  duplicate: { label: "Duplicate image", reason: "This image is a duplicate of: " },
  guidelines: {
    label: "Site-guideline violation",
    reason: "This image does not satisfy site guidelines because: ",
  },
  misleading: {
    label: "Incorrect or misleading content",
    reason: "This image contains incorrect or misleading content: ",
  },
  custom: { label: "Custom reason", reason: "" },
};

function Submission({ id }: { id: number }) {
  const auth = useAuth();
  const { mutate } = useSWRConfig();
  const [open, setOpen] = useState(false);
  const [template, setTemplate] = useState<keyof typeof templates>("custom");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const length = Array.from(reason.trim()).length;
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!busy) {
          setOpen(next);
          setError("");
        }
      }}
    >
      <DialogTrigger render={<Button variant="outline" size="sm" />}>
        Request deletion
      </DialogTrigger>
      <ResponsiveDialogContent>
        <DialogHeader>
          <DialogTitle>Request deletion</DialogTitle>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={async (event) => {
            event.preventDefault();
            if (!length || length > 2000 || busy) return;
            setBusy(true);
            setError("");
            try {
              await api.post(
                `/image/${id}/deletion`,
                { reason: reason.trim() },
                { headers: { Authorization: "Session " + auth.session } },
              );
              setOpen(false);
              setReason("");
              setTemplate("custom");
              await invalidateDeletionCaches(mutate);
            } catch (e) {
              setError(formatError(e));
              if (isAxiosError(e) && [401, 403].includes(e.response?.status ?? 0)) {
                toast.error(formatError(e));
                await auth.mutate();
              }
              if (isAxiosError(e) && e.response?.status === 409) {
                toast.error(formatError(e));
                await invalidateDeletionCaches(mutate);
              }
            } finally {
              setBusy(false);
            }
          }}
        >
          <Field>
            <FieldLabel>Reason template</FieldLabel>
            <Select
              value={template}
              onValueChange={(value) => {
                if (value) {
                  setTemplate(value);
                  setReason(templates[value].reason);
                }
              }}
              disabled={busy}
            >
              <SelectTrigger aria-label="Reason template">
                <SelectValue>
                  {(value) => templates[value as keyof typeof templates].label}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {Object.entries(templates).map(([key, item]) => (
                  <SelectItem key={key} value={key}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field>
            <FieldLabel htmlFor="deletion-reason">Reason</FieldLabel>
            <Textarea
              id="deletion-reason"
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              disabled={busy}
              rows={5}
              required
              aria-describedby="deletion-reason-length"
            />
            <p id="deletion-reason-length" className="text-xs text-muted-foreground">
              {length}/2,000 characters. You can edit any template and add an original image URL.
            </p>
          </Field>
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}
          <DialogFooter>
            <Button type="submit" disabled={busy || !length || length > 2000}>
              {busy ? "Submitting…" : "Submit request"}
            </Button>
            <ResponsiveDialogCancel disabled={busy} />
          </DialogFooter>
        </form>
      </ResponsiveDialogContent>
    </Dialog>
  );
}

export default function DeletionRequests({ id, deleted }: { id: number; deleted: boolean }) {
  const auth = useAuth();
  const [open, setOpen] = useState(false);
  const [offset, setOffset] = useState(0);
  const { data, error, isLoading, mutate } = useSWR<{ items: Request[]; total: number }>(
    open ? { id: "deletion-history", args: [id, offset], tags: ["image", "deletion"] } : null,
    async () =>
      (await api.get(`/image/${id}/deletion`, { params: { offset, limit: 10 } })).data.data,
    { shouldRetryOnError: false },
  );
  const canSubmit = Boolean(
    auth.session && !auth.isLoading && (auth.data?.permission ?? 0) & 4 && !deleted,
  );
  return (
    <Card className="mt-3 rounded-md py-2">
      <CardContent className="px-2">
        <Collapsible open={open} onOpenChange={setOpen}>
          <CollapsibleTrigger
            render={
              <Button variant="ghost" className="w-full">
                Deletion requests
                <ChevronDownIcon className="ml-auto transition-all group-data-panel-open/button:rotate-180" />
              </Button>
            }
          />
          <CollapsibleContent className="space-y-4 px-2 pb-2">
            {isLoading && <p role="status">Loading deletion requests…</p>}
            {error && (
              <div role="alert" className="space-y-2">
                <p>{formatError(error)}</p>
                <Button variant="outline" onClick={() => void mutate()}>
                  Retry
                </Button>
              </div>
            )}
            {data && !error && (
              <div className='my-2'>
                {!data.items.length && (
                  <p className="text-sm text-muted-foreground">No deletion requests.</p>
                )}
                <ul className="divide-y">
                  {data.items.map((request) => (
                    <li key={request.id} className="space-y-2 py-3 text-sm">
                      <div className="flex items-baseline gap-2">
                        <RequestStatusBadge status={request.status} />
                        <p className="min-w-0 flex-1 whitespace-pre-wrap break-words">
                          {request.reason}
                        </p>
                      </div>
                      <div className="flex flex-wrap items-center gap-x-1 gap-y-1 text-xs text-muted-foreground">
                        by
                        <Link
                          className="underline underline-offset-4"
                          to={`/user/${request.requester.id}`}
                        >
                          {request.requester.username}
                        </Link>
                        at
                        <time dateTime={request.createdAt}>
                          {new Date(request.createdAt).toLocaleString()}
                        </time>
                      </div>
                      {request.processedAt && (
                        <p className="text-xs text-muted-foreground">
                          Processed{" "}
                          <time dateTime={request.processedAt}>
                            {new Date(request.processedAt).toLocaleString()}
                          </time>
                        </p>
                      )}
                    </li>
                  ))}
                </ul>
                {data.total > 10 && (
                  <Pagination
                    count={Math.ceil(data.total / 10)}
                    page={offset / 10 + 1}
                    onChange={(_, page) => setOffset((page - 1) * 10)}
                  />
                )}
              </div>
            )}
            {canSubmit && (
              <div className="flex justify-end pt-4">
                <Submission id={id} />
              </div>
            )}
          </CollapsibleContent>
        </Collapsible>
      </CardContent>
    </Card>
  );
}
