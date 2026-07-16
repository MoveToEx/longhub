import { Dialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/shared/components/ui/dialog";
import { Dialog as BaseDialog } from '@base-ui/react';
import { Info, Trash } from "lucide-react";
import { Button } from "@/shared/components/ui/button";
import { useState } from "react";
import { Spinner } from "@/shared/components/ui/spinner";
import api from "@/shared/lib/axios";
import { formatError } from "@/shared/lib/utils";
import { toast } from "sonner";
import usePasskey from "../hooks/use-passkey";

type DialogPayload = {
  id: string,
  name: string
};

export default function DeletePasskeyDialog({
  handle
}: {
  handle: BaseDialog.Handle<DialogPayload>
}) {
  const [loading, setLoading] = useState(false);
  const { mutate } = usePasskey();

  const submit = async (id?: string) => {
    if (!id) return;

    setLoading(true);

    try {
      const urlSafe = id.replace(/\+/g, '-')
        .replace(/\//g, '_');
      await api.delete(`/user/webauthn/${urlSafe}`);
      await mutate();
      handle.close();
      toast.success('Deleted');
    }
    catch (e) {
      toast.error(formatError(e));
    }
    finally {
      setLoading(false);
    }
  }

  return (
    <Dialog<DialogPayload> handle={handle}>
      {({ payload }) => {
        return (
          <DialogContent>
            <DialogHeader>
              <DialogTitle className='text-xl'>
                Confirm deletion
              </DialogTitle>
            </DialogHeader>

            <div>
              You're about to delete Passkey <span className='font-mono p-1 border border-accent-foreground bg-accent rounded-md'>{payload?.name}</span>.
              <br />
              This action cannot be undone.
            </div>

            <span className='flex flex-row items-center gap-2 text-muted-foreground text-xs'>
              <Info size={12} className='inline' /> Don't forget to remove this key at your password manager
            </span>

            <DialogFooter>
              <Button
                variant='destructive'
                disabled={loading}
                onClick={() => submit(payload?.id)}>
                {loading && <Spinner />}
                {loading || <Trash />} Delete
              </Button>
              <DialogClose
                disabled={loading}
                render={
                  <Button variant='outline'>
                    Cancel
                  </Button>
                } />
            </DialogFooter>
          </DialogContent>
        )
      }}
    </Dialog>
  )
}
