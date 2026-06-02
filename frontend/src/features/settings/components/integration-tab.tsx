import { Button } from "@/shared/components/ui/button";
import { Dialog as BaseDialog } from '@base-ui/react';
import { Spinner } from "@/shared/components/ui/spinner";
import { Edit, EllipsisVertical, Key, Plus, Trash } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/shared/components/ui/dropdown-menu";
import useAppKeys from "@/features/settings/hooks/use-appkeys";
import EditAppkeyDialog from "@/features/settings/components/appkey-edit-dialog";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/shared/components/ui/empty";
import DeleteAppkeyDialog from "@/features/settings/components/appkey-delete-dialog";
import AddAppkeyDialog from "@/features/settings/components/appkey-add-dialog";
import ShowAppkeyDialog from "@/features/settings/components/appkey-show-dialog";

type DialogPayload = {
  id: number,
  label: string
};

const addHandle = BaseDialog.createHandle<void>();
const deleteHandle = BaseDialog.createHandle<DialogPayload>();
const editHandle = BaseDialog.createHandle<DialogPayload & { permission: number }>();
const showHandle = BaseDialog.createHandle<string>();

function AppKeySection() {
  const { data, isLoading } = useAppKeys();

  return (
    <div className='flex flex-col items-start gap-1'>
      <AddAppkeyDialog handle={addHandle} onComplete={val => showHandle.openWithPayload(val)} />
      <EditAppkeyDialog handle={editHandle} />
      <DeleteAppkeyDialog handle={deleteHandle} />
      <ShowAppkeyDialog handle={showHandle} />

      <span className='text-lg'>App keys</span>

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
                <Key />
              </EmptyMedia>
              <EmptyTitle>
                No app key yet
              </EmptyTitle>
              <EmptyDescription>
                Integrate LONG Hub into your app with an app key.
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
                {it.label}
              </span>
              <span className='text-xs text-muted-foreground flex flex-row justify-start gap-1'>
                <span>
                  Created at {new Date(it.createdAt).toLocaleDateString()}
                </span>
                <span>
                  •
                </span>
                {it.lastActivatedAt === null && (
                  <span>Never used</span>
                )}
                {it.lastActivatedAt !== null && (
                  <span>
                    Last used at {new Date(it.lastActivatedAt).toLocaleDateString()}
                  </span>
                )}
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
        <BaseDialog.Trigger
          handle={addHandle}
          render={
            <Button>
              <Plus />
              Add
            </Button>
          } />
      </div>
    </div>
  )
}

export default function IntegrationTab() {
  return (
    <div className='w-full flex flex-col items-center'>
      <div className='w-full md:w-1/2 flex flex-col'>
        <AppKeySection />

      </div>
    </div>
  )
}