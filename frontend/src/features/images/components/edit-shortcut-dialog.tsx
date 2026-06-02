import { LogIn } from "lucide-react";
import { Spinner } from "@/shared/components/ui/spinner";
import { useForm, Controller } from 'react-hook-form'
import z from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/shared/components/ui/field"
import { Input } from "@/shared/components/ui/input"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogClose, DialogDescription } from "@/shared/components/ui/dialog";
import { Button } from "@/shared/components/ui/button";
import api from "@/shared/lib/axios";
import { toast } from "sonner";
import { AxiosError } from "axios";
import { useEffect, useState } from "react";
import { mutate } from "@/shared/lib/swr";
import { useAppDispatch, useAppSelector } from "@/shared/hooks/use-redux";
import { closeEditShortcutDialog } from "@/features/images/state/edit-shortcut-dialog-slice";


const schema = z.object({
  shortcut: z.string(),
})

export default function EditShortcutDialog() {
  const dispatch = useAppDispatch();
  const open = useAppSelector(state => state.editShortcutDialog.open);
  const payload = useAppSelector(state => state.editShortcutDialog.payload);
  const [loading, setLoading] = useState(false);

  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      shortcut: ''
    }
  });

  useEffect(() => {
    if (!payload) {
      form.reset({ shortcut: '' });
      return;
    }

    if (payload.shortcut !== form.getValues('shortcut')) {
      form.reset({ shortcut: payload.shortcut });
    }
  }, [form, payload]);

  const onSubmit = async (data: z.infer<typeof schema>) => {
    if (!payload) return;

    setLoading(true);
    try {
      const response = await api.patch(`/favorite/${payload.id}`, data);

      if (response.status === 204) {
        mutate('favorite');
        toast.success('Saved');
        dispatch(closeEditShortcutDialog());
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
    <Dialog
      open={open}
      onOpenChange={val => {
        if (!val) {
          dispatch(closeEditShortcutDialog());
        }
      }}>
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
    </Dialog>
  )
}
