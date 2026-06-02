import { Dialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/shared/components/ui/dialog";
import { Dialog as BaseDialog } from '@base-ui/react';
import { Trash } from "lucide-react";
import { Button } from "@/shared/components/ui/button";
import { useState } from "react";
import { Spinner } from "@/shared/components/ui/spinner";
import api from "@/shared/lib/axios";
import { AxiosError } from "axios";
import { toast } from "sonner";
import useAppKeys from "../hooks/use-appkeys";

type DialogPayload = {
  id: number,
  label: string
};

export default function DeleteAppkeyDialog({
  handle
}: {
  handle: BaseDialog.Handle<DialogPayload>
}) {
  const [loading, setLoading] = useState(false);
  const { mutate } = useAppKeys();

  const submit = async (id?: number) => {
    if (!Number.isSafeInteger(id)) return;

    setLoading(true);

    try {
      await api.delete(`/user/appkey/${id}`);
      await mutate();
      handle.close();
      toast.success('Deleted');
    }
    catch (e) {
      if (e instanceof AxiosError) {
        toast.error(e.response?.data.error);
      }
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
              You're about to delete {payload?.label}.
              <br />
              This action cannot be undone.
            </div>

            <DialogFooter>
              <Button
                variant='destructive'
                disabled={loading}
                onClick={() => submit(Number(payload?.id))}>
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