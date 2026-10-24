import { useId, useState } from "react";
import { UserCog } from "lucide-react";
import { isAxiosError } from "axios";
import { useSWRConfig } from "swr";
import { toast } from "sonner";
import useAuth from "@/features/auth/hooks/use-auth";
import PermissionTransferList from "@/features/settings/components/permission-transfer-list";
import { Button } from "@/shared/components/ui/button";
import {
  Dialog,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/shared/components/ui/dialog";
import { Field, FieldDescription, FieldLabel } from "@/shared/components/ui/field";
import { Input } from "@/shared/components/ui/input";
import {
  ResponsiveDialogCancel,
  ResponsiveDialogContent,
} from "@/shared/components/ui/responsive-dialog";
import { Spinner } from "@/shared/components/ui/spinner";
import { ADMIN_PERMISSION, hasRootPermission } from "@/shared/lib/const";
import { formatError } from "@/shared/lib/utils";
import { adminGraphQL } from "../lib/graphql";
import { UPDATE_USER, type AdminUser } from "../lib/types";

const baseEditablePermissions = 1 | (1 << 1) | (1 << 2);

export default function UserManagementDialog({ user }: { user: AdminUser }) {
  const id = useId();
  const auth = useAuth();
  const isRoot = hasRootPermission(auth.data?.permission);
  const protectedAccount = hasRootPermission(user.permission) && !isRoot;
  const editablePermissions = baseEditablePermissions | (isRoot ? ADMIN_PERMISSION : 0);
  const { mutate } = useSWRConfig();
  const [open, setOpen] = useState(false);
  const [email, setEmail] = useState(user.email);
  const [password, setPassword] = useState("");
  const [permission, setPermission] = useState(user.permission & editablePermissions);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (busy) return;
        setOpen(next);
        setPassword("");
        setError("");
        if (next) {
          setEmail(user.email);
          setPermission(user.permission & editablePermissions);
        }
      }}
    >
      <DialogTrigger render={<Button variant="outline" size="sm" aria-label={`Manage ${user.username}`} disabled={protectedAccount} title={protectedAccount ? "Only Root users can manage Root accounts" : undefined} />}>
        <UserCog aria-hidden="true" /> Manage
      </DialogTrigger>
      <ResponsiveDialogContent>
        <DialogHeader>
          <DialogTitle>Manage user</DialogTitle>
          <DialogDescription>{user.username} · #{user.id}</DialogDescription>
        </DialogHeader>
        <form
          className="space-y-5"
          onSubmit={async (event) => {
            event.preventDefault();
            if (busy) return;
            if (new TextEncoder().encode(password).length > 72) {
              setError("Password must be at most 72 bytes.");
              return;
            }
            setBusy(true);
            setError("");
            try {
              await adminGraphQL(
                UPDATE_USER,
                {
                  id: user.id,
                  input: {
                    email: email.trim(),
                    permission: permission & editablePermissions,
                    ...(password ? { password } : {}),
                  },
                },
                auth.session,
              );
              setPassword("");
              setOpen(false);
              toast.success("User updated");
              // Keep cached identity and rows so admin queries stay mounted during revalidation.
              await mutate(
                (key) => Array.isArray(key)
                  ? key[0] === "admin" && key[2] === "users"
                  : Boolean(key && typeof key === "object" && "tags" in key &&
                    Array.isArray(key.tags) && key.tags.includes("user")),
              );
            } catch (e) {
              setError(formatError(e));
              if (isAxiosError(e) && [401, 403].includes(e.response?.status ?? 0)) {
                await auth.mutate();
              }
            } finally {
              setBusy(false);
            }
          }}
        >
          <Field>
            <FieldLabel htmlFor={`${id}-email`}>Email</FieldLabel>
            <Input
              id={`${id}-email`}
              type="email"
              autoComplete="off"
              required
              maxLength={255}
              value={email}
              disabled={busy}
              onChange={(event) => setEmail(event.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor={`${id}-password`}>New password</FieldLabel>
            <Input
              id={`${id}-password`}
              type="password"
              autoComplete="new-password"
              aria-describedby={`${id}-password-help`}
              value={password}
              disabled={busy}
              onChange={(event) => setPassword(event.target.value)}
            />
            <FieldDescription id={`${id}-password-help`}>
              Leave blank to keep the current password.
            </FieldDescription>
          </Field>
          <fieldset disabled={busy} className="space-y-3">
            <legend className="mb-3 font-medium">Permissions</legend>
            <PermissionTransferList
              mask={editablePermissions}
              value={permission & editablePermissions}
              onValueChange={(value) => setPermission(value & editablePermissions)}
              disabled={busy}
            />
          </fieldset>
          {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
          <DialogFooter>
            <ResponsiveDialogCancel disabled={busy} />
            <Button type="submit" disabled={busy}>
              {busy && <Spinner />} Save changes
            </Button>
          </DialogFooter>
        </form>
      </ResponsiveDialogContent>
    </Dialog>
  );
}
