import { LogIn } from "lucide-react";
import { Spinner } from "@/components/ui/spinner";
import { useForm, Controller } from 'react-hook-form'
import z from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogClose, DialogDescription } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Dialog as BaseDialog } from "@base-ui/react";
import api from "@/lib/axios";
import { toast } from "sonner";
import { AxiosError } from "axios";
import { useEffect, useState } from "react";
import { mutate } from "@/lib/swr";
import _ from "lodash";


const schema = z.object({
  shortcut: z.string(),
})

export type Payload = {
  shortcut: string,
  id: number,
}

export default function EditShortcutDialog({
  handle
}: {
  handle: BaseDialog.Handle<Payload>
}) {
  const [loading, setLoading] = useState(false);

  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      shortcut: ''
    }
  });

  return (
    <Dialog<Payload>
      handle={handle}
      onOpenChangeComplete={() => {
        form.reset();
      }}>
      {function Content({ payload }) {
        useEffect(() => {
          if (!payload) return;

          if (!_.isEqual(payload, form.getValues())) {
            form.reset(payload);
          }
        }, [payload]);
        const onSubmit = async (data: z.infer<typeof schema>) => {
          setLoading(true);
          try {
            const response = await api.patch(`/favorite/${payload?.id}`, data);

            if (response.status === 204) {
              mutate('favorite');
              toast.success('Saved');
              handle.close();
            }
          }
          catch (e) {
            if (e instanceof AxiosError) {
              form.setError('root', {
                type: 'custom',
                message: e.response?.data.error
              })
            }
          }
          finally {
            setLoading(false);
          }
        }
        return (
          <DialogContent>
            <DialogHeader>
              <DialogTitle>
                <span className='text-xl'>Customize shortcut</span>
              </DialogTitle>

              <DialogDescription>
                Favorited images can be found in Quick Search by entering shortcuts
              </DialogDescription>
            </DialogHeader>

            <form onSubmit={form.handleSubmit(onSubmit)}>
              <FieldGroup className='mb-4'>
                <Controller
                  name='shortcut'
                  control={form.control}
                  render={({ field, fieldState }) => (
                    <Field data-invalid={fieldState.invalid}>
                      <FieldLabel htmlFor="form-favorite-shortcut">
                        Shortcut
                      </FieldLabel>
                      <Input
                        {...field}
                        id="form-favorite-shortcut"
                      />
                      {fieldState.invalid && (
                        <FieldError errors={[fieldState.error]} />
                      )}
                    </Field>
                  )} />
              </FieldGroup>


              <DialogFooter>
                <Button type='submit' disabled={loading}>
                  {loading && <Spinner />}
                  {loading || <LogIn />}
                  Confirm
                </Button>
                <DialogClose render={<Button variant='outline'>Cancel</Button>} />
              </DialogFooter>
            </form>
          </DialogContent>
        );
      }}

    </Dialog>
  )
}
