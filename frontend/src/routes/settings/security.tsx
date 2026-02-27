import { Button } from "@/components/ui/button";
import { Dialog as BaseDialog } from '@base-ui/react';
import { Spinner } from "@/components/ui/spinner";
import usePasskey from "@/hooks/server/use-passkey"
import api from "@/lib/axios";
import type { Wrapped } from "@/lib/types";
import { base64ToArray } from "@/lib/utils";
import { AxiosError } from "axios";
import { Edit, EllipsisVertical, Key, Plus, ScanFace, Trash } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import DeletePasskeyDialog from "@/components/dialogs/passkey-delete";
import EditPasskeyDialog from "@/components/dialogs/passkey-edit";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";

type WebAuthnCreationResponse = {
  publicKey: {
    rp: {
      id: string,
      name: string,
    },
    user: {
      id: string,
      name: string,
      displayName: string,
    },
    challenge: string,
    pubKeyCredParams: {
      type: 'public-key',
      alg: number
    }[],
    timeout: number,
    authenticatorSelection: {
      residentKey: 'required',
      userVerification: 'required'
    }
  }
}

type DialogPayload = {
  id: string,
  name: string
};

type ValidateResponse = {
  id: string;
};

const deleteHandle = BaseDialog.createHandle<DialogPayload>();
const editHandle = BaseDialog.createHandle<DialogPayload>();

function PasskeySection() {
  const { data, isLoading, mutate } = usePasskey();
  const [loading, setLoading] = useState(false);

  const add = async () => {
    setLoading(true);

    try {
      const data = await api.post<Wrapped<WebAuthnCreationResponse>>('/user/webauthn/new');

      const { publicKey } = data.data.data;

      const assertion = await navigator.credentials.create({
        publicKey: {
          ...publicKey,
          challenge: base64ToArray(publicKey.challenge),
          user: {
            ...publicKey.user,
            id: base64ToArray(publicKey.user.id),
          }
        }
      }) as PublicKeyCredential;

      if (assertion === null) {
        toast.info('Cancelled');
        return;
      }

      const valResp = await api.post<Wrapped<ValidateResponse>>('/user/webauthn/validate', assertion);

      const id = valResp.data.data.id;

      await mutate();

      toast.success('Added');

      editHandle.openWithPayload({ id, name: 'Unnamed' });
    }
    catch (e) {
      if (e instanceof AxiosError) {
        toast.error(e.response?.data.error)
      }
      else if (e instanceof Error) {
        console.log(e.message);
      }
    }
    finally {
      setLoading(false);
    }
  }

  return (
    <div className='flex flex-col items-start gap-1'>
      <DeletePasskeyDialog handle={deleteHandle} />
      <EditPasskeyDialog handle={editHandle} />
      <span className='text-lg'>Passkeys</span>

      {isLoading && (
        <div className='h-12 flex flex-row items-center gap-2'>
          <Spinner />
          Loading...
        </div>
      )}

      <div className='w-full min-h-12 flex flex-col gap-2'>
        {data?.length === 0 && (
          <Empty className='h-16'>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <ScanFace />
              </EmptyMedia>
              <EmptyTitle>
                No passkey yet
              </EmptyTitle>
              <EmptyDescription>
                You haven't added any passkeys.
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        )}
        {data?.map(it => (
          <div key={it.id} className='flex flex-row gap-2 border border-accent p-4 rounded-md'>
            <div>
              <Key size={16} />
            </div>
            <div className='flex flex-col flex-1 gap-1'>
              <span>
                {it.name}
              </span>
              <span className='text-xs text-muted-foreground'>
                Created at {new Date(it.createdAt).toLocaleDateString()}
              </span>
            </div>
            <div>
              <DropdownMenu>
                <DropdownMenuTrigger render={
                  <Button variant='outline' size='icon'>
                    <EllipsisVertical />
                  </Button>
                } />
                <DropdownMenuContent>
                  <DropdownMenuItem
                    onClick={() => {
                      editHandle.openWithPayload(it);
                    }}>
                    <Edit /> Edit
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    variant='destructive'
                    onClick={() => {
                      deleteHandle.openWithPayload(it);
                    }}>
                    <Trash /> Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        ))}
      </div>

      <div className='w-full flex flex-row justify-end items-center gap-2'>
        <Button disabled={loading} onClick={() => add()}>
          {loading && <Spinner />}
          {loading || <Plus />}
          Add
        </Button>
      </div>
    </div>
  )
}

export default function SecurityTab() {
  return (
    <div className='w-full flex flex-col items-center'>
      <div className='w-full md:w-1/2 flex flex-col'>
        <PasskeySection />

      </div>
    </div>
  )
}