import { Dialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/shared/components/ui/dialog";
import { Dialog as BaseDialog } from '@base-ui/react';
import { Send } from "lucide-react";
import { Button } from "@/shared/components/ui/button";
import { useState } from "react";
import { Spinner } from "@/shared/components/ui/spinner";
import api from "@/shared/lib/axios";
import { formatError } from "@/shared/lib/utils";
import { toast } from "sonner";
import { Controller, useForm } from "react-hook-form";
import z from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/shared/components/ui/field";
import { Input } from "@/shared/components/ui/input";
import useAuth from "@/features/auth/hooks/use-auth";
import PermissionTransferList from "@/features/settings/components/permission-transfer-list";
import type { Wrapped } from "@/shared/lib/types";
import useAppKeys from "../hooks/use-appkeys";

const schema = z.object({
  label: z.string(),
  permission: z.number().int(),
});

export default function AddAppkeyDialog({
  handle,
  onComplete
}: {
  handle: BaseDialog.Handle<void>,
  onComplete: (key: string) => void,
}) {
  const [loading, setLoading] = useState(false);
  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      label: '',
      permission: 0,
    }
  });
  const { data: user } = useAuth();
  const { mutate } = useAppKeys();

  const submit = async (data: z.infer<typeof schema>) => {
    setLoading(true);

    try {
      const response = await api.post(`/user/appkey`, data);

      const { data: { key } }: Wrapped<{ key: string }> = response.data;
      await mutate();
      handle.close();
      toast.success('Added');
      onComplete(key);
    }
    catch (e) {
      toast.error(formatError(e));
    }
    finally {
      setLoading(false);
    }
  }

  return (
    <Dialog handle={handle} onOpenChangeComplete={() => form.reset()}>
      <DialogContent className='sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle className='text-xl'>
            New App key
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={form.handleSubmit(submit)}>
          <FieldGroup>
            <Controller
              name='label'
              control={form.control}
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel htmlFor="form-appkey-label">
                    Label
                  </FieldLabel>
                  <Input
                    {...field}
                    id="form-appkey-label"
                  />
                  {fieldState.invalid && (
                    <FieldError errors={[fieldState.error]} />
                  )}
                </Field>
              )} />

            <Controller
              name='permission'
              control={form.control}
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <PermissionTransferList
                    {...field}
                    onValueChange={val => form.setValue('permission', val)}
                    mask={user?.permission ?? 0} />
                  {fieldState.invalid && (
                    <FieldError errors={[fieldState.error]} />
                  )}
                </Field>
              )} />
          </FieldGroup>

          <DialogFooter className='mt-4'>
            <Button
              type='submit'
              disabled={loading}>
              {loading && <Spinner />}
              {loading || <Send />} Submit
            </Button>
            <DialogClose
              disabled={loading}
              render={
                <Button variant='outline'>
                  Cancel
                </Button>
              } />
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
