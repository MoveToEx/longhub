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
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose, DialogTrigger } from "@/components/ui/dialog";
import { Dialog as BaseDialog } from '@base-ui/react'
import { Button } from "@/components/ui/button";
import api from "@/lib/axios";
import { toast } from "sonner";
import { AxiosError } from "axios";
import { useState } from 'react'
import { User } from 'lucide-react'
import { Spinner } from '../ui/spinner'

const registerSchema = z.object({
  email: z.email(),
  username: z.string()
    .min(6, 'Username should be no shorter than 6 characters')
    .max(32, 'Username should be no longer than 32 characters')
    .regex(/^[a-zA-Z0-9]+$/, 'Only digits and letters are allowed'),
  password: z.string()
    .min(8, 'Password should be at lease 8 characters')
    .max(128, 'Password should be at most 128 characters'),
  confirmPassword: z.string()
    .min(8, 'Password should be at lease 8 characters')
    .max(128, 'Password should be at most 128 characters'),
});

const registerHandle = BaseDialog.createHandle();

export default function RegisterDialog() {
  const [loading, setLoading] = useState(false);

  const form = useForm<z.infer<typeof registerSchema>>({
    resolver: zodResolver(registerSchema),
    defaultValues: {
      email: '',
      username: '',
      password: '',
      confirmPassword: '',
    },
    disabled: loading
  });

  const onSubmit = async (data: z.infer<typeof registerSchema>) => {
    if (data.confirmPassword !== data.password) {
      form.setError('confirmPassword', {
        type: 'validate',
        message: 'Passwords do not match'
      });
      return;
    }
    
    setLoading(true);

    try {
      const response = await api.post('/auth/register', data);

      if (response.status === 204) {
        toast.success('Successfully signed up');
        registerHandle.close();
      }
    }
    catch (e) {
      if (e instanceof AxiosError) {
        form.setError('root', {
          type: 'custom',
          message: e.response?.data.error
        });
      }
    }
    finally {
      setLoading(false);
    }
  }

  return (
    <Dialog
      handle={registerHandle}
      onOpenChangeComplete={() => {
        form.reset();
      }}>
      <DialogTrigger render={
        <Button variant='link'>
          Register
        </Button>
      } />
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            <span className='text-xl'>Register</span>
          </DialogTitle>
          <DialogDescription>
            <span className='text-muted-foreground'>Where the mothercide sets out</span>
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={form.handleSubmit(onSubmit)}>
          <FieldGroup>
            <Controller
              name='email'
              control={form.control}
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel htmlFor="form-register-email">
                    Email
                  </FieldLabel>
                  <Input
                    {...field}
                    id="form-register-email"
                    placeholder="user@example.com"
                    autoComplete="email"
                  />
                  {fieldState.invalid && (
                    <FieldError errors={[fieldState.error]} />
                  )}
                </Field>
              )} />
            <Controller
              name='username'
              control={form.control}
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel htmlFor="form-register-username">
                    Username
                  </FieldLabel>
                  <Input
                    {...field}
                    id="form-register-username"
                    placeholder="user@example.com"
                    autoComplete="username"
                  />
                  {fieldState.invalid && (
                    <FieldError errors={[fieldState.error]} />
                  )}
                </Field>
              )} />
            <Controller
              name='password'
              control={form.control}
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel htmlFor="form-register-password">
                    Password
                  </FieldLabel>
                  <Input
                    {...field}
                    id="form-register-password"
                    placeholder="••••••••"
                    autoComplete='new-password'
                    type='password'
                  />
                  {fieldState.invalid && (
                    <FieldError errors={[fieldState.error]} />
                  )}
                </Field>
              )} />
            <Controller
              name='confirmPassword'
              control={form.control}
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel htmlFor="form-register-confirm-password">
                    Confirm Password
                  </FieldLabel>
                  <Input
                    {...field}
                    id="form-register-confirm-password"
                    placeholder="••••••••"
                    autoComplete="new-password"
                    type='password'
                  />
                  {fieldState.invalid && (
                    <FieldError errors={[fieldState.error]} />
                  )}
                </Field>
              )} />
          </FieldGroup>

          <div className='flex flex-row justify-end'>
            <span>
              Already have an account?
              <DialogClose render={
                <Button variant='link'>
                  Go back to login
                </Button>
              } />
            </span>
          </div>

          <DialogFooter>
            <Button type='submit' disabled={loading}>
              {loading || <User />}
              {loading && <Spinner />}
              Register
            </Button>
            <DialogClose render={<Button variant='outline'>Cancel</Button>} />
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}