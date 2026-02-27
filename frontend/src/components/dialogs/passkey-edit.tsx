import { Dialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Dialog as BaseDialog } from '@base-ui/react';
import { Send } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useState } from "react";
import { Spinner } from "@/components/ui/spinner";
import api from "@/lib/axios";
import { AxiosError } from "axios";
import { toast } from "sonner";
import { mutate } from "@/lib/swr";
import { Controller, useForm } from "react-hook-form";
import z from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

type DialogPayload = {
  id: string,
  name: string
};

const schema = z.object({
  name: z.string(),
})

export default function EditPasskeyDialog({
  handle
}: {
  handle: BaseDialog.Handle<DialogPayload>
}) {
  const [loading, setLoading] = useState(false);
  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: ''
    }
  });

  return (
    <Dialog<DialogPayload> handle={handle} onOpenChangeComplete={() => form.reset()}>
      {({ payload }) => {
        const submit = async (data: z.infer<typeof schema>) => {
          if (!payload) return;

          setLoading(true);

          try {
            const urlSafe = payload.id.replace(/\+/g, '-')
              .replace(/\//g, '_');
            await api.patch(`/user/webauthn/${urlSafe}`, data);
            await mutate('passkey');
            handle.close();
            toast.success('Saved');
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
          <DialogContent>
            <DialogHeader>
              <DialogTitle className='text-xl'>
                Edit Passkey
              </DialogTitle>
            </DialogHeader>

            <form onSubmit={form.handleSubmit(submit)}>
              <FieldGroup>
                <Controller
                  name='name'
                  control={form.control}
                  render={({ field, fieldState }) => (
                    <Field data-invalid={fieldState.invalid}>
                      <FieldLabel htmlFor="form-passkey-name">
                        Name
                      </FieldLabel>
                      <Input
                        {...field}
                        id="form-passkey-name"
                        placeholder="Yubico 5C"
                      />
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
        )
      }}
    </Dialog>
  )
}