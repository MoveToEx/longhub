import { Dialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Dialog as BaseDialog } from '@base-ui/react';
import { Send } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useEffect, useMemo, useState } from "react";
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
import useAuth from "@/hooks/server/use-auth";
import PermissionTransferList from "../permission-transfer-list";
import _ from 'lodash';

const schema = z.object({
  label: z.string(),
  permission: z.number().int(),
});

type Payload = {
  id: number,
  label: string,
  permission: number,
};

export default function EditAppkeyDialog({
  handle
}: {
  handle: BaseDialog.Handle<Payload>
}) {
  const [loading, setLoading] = useState(false);

  const { data: user } = useAuth();

  return (
    <Dialog<Payload> handle={handle}>
      {function Content({ payload }) {
        const form = useForm<z.infer<typeof schema>>({
          resolver: zodResolver(schema),
          defaultValues: useMemo(() => {
            if (!payload) {
              return {
                label: '',
                permission: 0,
              }
            }
            return payload;
          }, [payload])
        });

        const submit = async (data: z.infer<typeof schema>) => {
          setLoading(true);

          try {
            await api.patch(`/user/appkey/${payload?.id}`, data);
            await mutate('appkey');
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

        useEffect(() => {
          if (!payload) return;

          if (!_.isEqual(payload, form.getValues())) {
            form.reset(payload);
          }
        }, [payload, form]);

        return (
          <DialogContent className='sm:max-w-xl'>
            <DialogHeader>
              <DialogTitle className='text-xl'>
                Edit App key
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
        )
      }}
    </Dialog>
  )
}