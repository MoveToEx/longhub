import { Dialog as BaseDialog } from "@base-ui/react";
import { Trash } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import useWebhooks from "@/features/settings/hooks/use-webhooks";
import { Button } from "@/shared/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/shared/components/ui/dialog";
import { Spinner } from "@/shared/components/ui/spinner";
import api from "@/shared/lib/axios";
import type { Webhook } from "@/shared/lib/types";
import { formatError } from "@/shared/lib/utils";

export default function DeleteWebhookDialog({
  handle,
}: {
  handle: BaseDialog.Handle<Webhook>,
}) {
  const [loading, setLoading] = useState(false);
  const { mutate } = useWebhooks();

  const submit = async (webhook?: Webhook) => {
    if (!webhook) return;
    setLoading(true);

    try {
      await api.delete(`/user/webhook/${webhook.id}`);
      await mutate();
      handle.close();
      toast.success('Webhook deleted');
    }
    catch (error) {
      toast.error(formatError(error));
    }
    finally {
      setLoading(false);
    }
  };

  return (
    <Dialog<Webhook> handle={handle}>
      {({ payload }) => (
        <DialogContent>
          <DialogHeader>
            <DialogTitle className='text-xl'>Delete webhook?</DialogTitle>
          </DialogHeader>

          <div>
            You're about to delete <span className='font-medium'>{payload?.label}</span>.
            <br />
            This action cannot be undone.
          </div>

          <DialogFooter>
            <Button variant='destructive' disabled={loading} onClick={() => submit(payload)}>
              {loading ? <Spinner /> : <Trash />}
              Delete webhook
            </Button>
            <DialogClose disabled={loading} render={<Button variant='outline' />}>
              Cancel
            </DialogClose>
          </DialogFooter>
        </DialogContent>
      )}
    </Dialog>
  );
}
