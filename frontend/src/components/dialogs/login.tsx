import { Key, LogIn } from "lucide-react";
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
import { Dialog as BaseDialog } from '@base-ui/react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import api from "@/lib/axios";
import { toast } from "sonner";
import { AxiosError } from "axios";
import { useLocalStorage } from "usehooks-ts";
import { useState } from "react";
import RegisterDialog from "./register";
import { base64ToArray } from "@/lib/utils";
import type { Wrapped } from "@/lib/types";
import { mutate } from "@/lib/swr";


const loginSchema = z.object({
  username: z.string(),
  password: z.string()
})

type LoginMethod = 'password' | 'webauthn';

type LoginResponse = {
  token: string;
}

type FormProps = {
  handle: BaseDialog.Handle<void>,
  onMethodChange: (val: LoginMethod) => void,
}

function PasswordLogin({
  handle,
  onMethodChange,
}: FormProps) {
  const [, setSession] = useLocalStorage('nmsl-session', '');
  const [loading, setLoading] = useState(false);

  const form = useForm<z.infer<typeof loginSchema>>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      username: '',
      password: ''
    }
  });

  const onSubmit = async (data: z.infer<typeof loginSchema>) => {
    setLoading(true);
    try {
      const response = await api.post<Wrapped<LoginResponse>>('/auth/login', data);

      if (response.status === 200) {
        setSession(response.data.data.token);
        toast.success('Successfully logged in');
        mutate('self');
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
    <form onSubmit={form.handleSubmit(onSubmit)}>
      <FieldGroup>
        <Controller
          name='username'
          control={form.control}
          render={({ field, fieldState }) => (
            <Field data-invalid={fieldState.invalid}>
              <FieldLabel htmlFor="form-login-username">
                Username/Email
              </FieldLabel>
              <Input
                {...field}
                id="form-login-username"
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
              <FieldLabel htmlFor="form-login-password">
                Password
              </FieldLabel>
              <Input
                {...field}
                id="form-login-password"
                placeholder="••••••••"
                autoComplete='current-password'
                type='password'
              />
              {fieldState.invalid && (
                <FieldError errors={[fieldState.error]} />
              )}
            </Field>
          )} />
      </FieldGroup>

      <div className='w-full flex flex-col mt-6 gap-2'>
        <Button variant='outline' onClick={() => onMethodChange('webauthn')}>
          <Key /> Login with passkey
        </Button>

        <div className='flex flex-row justify-end'>
          <span>
            No account yet?
            <RegisterDialog />
          </span>
        </div>
      </div>


      <DialogFooter>
        <Button type='submit' disabled={loading}>
          {loading && <Spinner />}
          {loading || <LogIn />}
          Login
        </Button>
        <DialogClose render={<Button variant='outline'>Cancel</Button>} />
      </DialogFooter>
    </form>
  )
}

type WebAuthnNewResponse = {
  publicKey: {
    challenge: string,
    timeout: number,
    rpId: string,
  }
}

function WebAuthnLogin({
  handle,
  onMethodChange,
}: FormProps) {
  const [, setSession] = useLocalStorage('nmsl-session', '');
  const [loading, setLoading] = useState(false);

  const onSubmit = async () => {
    setLoading(true);
    try {
      const data = await api.post<Wrapped<WebAuthnNewResponse>>('/auth/login/webauthn/new');

      const { publicKey } = data.data.data;

      const assertion = await navigator.credentials.get({
        publicKey: {
          ...publicKey,
          challenge: base64ToArray(publicKey.challenge),
          userVerification: 'required',
        }
      }) as PublicKeyCredential;

      if (assertion === null) {
        return;
      }

      const result = await api.post<Wrapped<LoginResponse>>('/auth/login/webauthn/validate', assertion);

      setSession(result.data.data.token);
      await mutate('self');
      toast.success('Successfully logged in');
      handle.close();
    }
    catch (e) {
      if (e instanceof AxiosError) {
        toast.error(e.response?.data.error)
      }
    }
    finally {
      setLoading(false);
    }
  }

  return (
    <div className='w-full flex flex-col gap-4'>
      <span className='text-center text-lg'>
        <Key className='inline' /> Proceed with Passkey
      </span>
      <Button onClick={() => onSubmit()} disabled={loading}>
        {loading && <Spinner />}
        Continue
      </Button>

      <Button variant='outline' onClick={() => onMethodChange('password')}>
        Back to password
      </Button>

    </div>
  )
}

export default function LoginDialog({
  handle
}: {
  handle: BaseDialog.Handle<void>
}) {
  const [method, setMethod] = useState<LoginMethod>('password');

  return (
    <Dialog handle={handle} onOpenChangeComplete={() => setMethod('password')}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            <span className='text-xl'>Login</span>
          </DialogTitle>
          <DialogDescription>
            <span className='text-muted-foreground'>Welcome back, Mother killer</span>
          </DialogDescription>
        </DialogHeader>

        {method === 'password' && <PasswordLogin handle={handle} onMethodChange={val => setMethod(val)} />}
        {method === 'webauthn' && <WebAuthnLogin handle={handle} onMethodChange={val => setMethod(val)} />}
      </DialogContent>
    </Dialog>
  )
}
