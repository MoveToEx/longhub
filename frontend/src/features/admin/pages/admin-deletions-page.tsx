import { useState } from "react";
import { Link } from "react-router";
import { isAxiosError } from "axios";
import { toast } from "sonner";
import { useSWRConfig } from "swr";
import useAuth from "@/features/auth/hooks/use-auth";
import { invalidateDeletionCaches } from "@/features/images/lib/deletion-cache";
import { Button } from "@/shared/components/ui/button";
import { formatError } from "@/shared/lib/utils";
import AdminTable, { type AdminColumn } from "../components/admin-table";
import { adminGraphQL } from "../lib/graphql";
import {
  DELETION_QUERY,
  MODERATE_DELETION,
  type AdminDeletion,
  type DeletionAction,
} from "../lib/types";

export function DeletionActions({ row }: { row: AdminDeletion }) {
  const auth = useAuth();
  const { mutate } = useSWRConfig();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const actions: DeletionAction[] =
    row.status === "APPROVED"
      ? []
      : row.status === "ACTIVE"
        ? ["ACCEPT", "REJECT"]
        : ["REOPEN", "ACCEPT"];
  const blocked = Boolean(row.imageDeletedAt);
  const submit = async (action: DeletionAction) => {
    setBusy(true);
    setError("");
    try {
      await adminGraphQL(
        MODERATE_DELETION,
        { id: row.id, action, expectedStatus: row.status },
        auth.session,
      );
      await invalidateDeletionCaches(mutate);
    } catch (e) {
      setError(formatError(e));
      toast.error(formatError(e));
      if (isAxiosError(e) && [401, 403].includes(e.response?.status ?? 0)) {
        await mutate((key) => Array.isArray(key) && key[0] === "admin", undefined, {
          revalidate: false,
        });
        await auth.mutate();
      } else {
        // A conflict means another moderator changed the request or image.
        await invalidateDeletionCaches(mutate);
      }
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="min-w-44 space-y-2">
      <div className="flex gap-2">
        {actions.map((action) => (
          <Button
            key={action}
            size="sm"
            variant="outline"
            disabled={busy || (action === "ACCEPT" && blocked)}
            onClick={() => void submit(action)}
          >
            {{ ACCEPT: "Accept", REJECT: "Reject", REOPEN: "Reopen" }[action]}
          </Button>
        ))}
      </div>
      {blocked && actions.includes("ACCEPT") && (
        <p className="max-w-60 text-xs text-muted-foreground">
          This image is permanently deleted. This request cannot be accepted.
        </p>
      )}
      {error && (
        <p role="alert" className="max-w-64 text-xs text-destructive">
          {error}
        </p>
      )}
    </div>
  );
}

const date = (value: string | null) =>
  value ? (
    <time dateTime={value} className="whitespace-nowrap">
      {new Date(value).toLocaleString()}
    </time>
  ) : (
    "—"
  );
const user = (value: AdminDeletion["moderator"]) =>
  value ? (
    <Link className="whitespace-nowrap underline underline-offset-4" to={`/user/${value.id}`}>
      {value.username} #{value.id}
    </Link>
  ) : (
    "—"
  );
const columns: AdminColumn<AdminDeletion>[] = [
  { id: "id", label: "ID", sortBy: "ID", render: (row) => row.id },
  {
    id: "image",
    label: "Image",
    render: (row) => (
      <Link className="underline underline-offset-4" to={`/image/${row.imageId}`}>
        #{row.imageId}
      </Link>
    ),
  },
  {
    id: "reason",
    label: "Reason",
    render: (row) => (
      <p className="min-w-56 max-w-96 whitespace-pre-wrap break-words">{row.reason}</p>
    ),
  },
  { id: "requester", label: "Requester", render: (row) => user(row.requester) },
  {
    id: "status",
    label: "Status",
    render: (row) => <span className="capitalize">{row.status.toLowerCase()}</span>,
  },
  { id: "created", label: "Created", sortBy: "CREATED_AT", render: (row) => date(row.createdAt) },
  { id: "moderator", label: "Moderator", render: (row) => user(row.moderator) },
  {
    id: "processed",
    label: "Processed",
    sortBy: "PROCESSED_AT",
    render: (row) => date(row.processedAt),
  },
  { id: "actions", label: "Actions", render: (row) => <DeletionActions row={row} /> },
];

export default function AdminDeletionsPage() {
  return (
    <AdminTable
      resource="deletionRequests"
      title="Deletion requests"
      searchPlaceholder="Search by request/image ID, requester, or reason…"
      query={DELETION_QUERY}
      columns={columns}
    />
  );
}
