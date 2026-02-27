import { Dialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Dialog as BaseDialog } from '@base-ui/react';
import { Button } from "@/components/ui/button";
import { Input } from "../ui/input";
import { Check, Copy, Info } from "lucide-react";
import { useState } from "react";

export default function ShowAppkeyDialog({
  handle
}: {
  handle: BaseDialog.Handle<string>
}) {
  return (
    <Dialog handle={handle}>
      {function Content({ payload }) {
        const [copied, setCopied] = useState(false);

        return (
          <DialogContent className='sm:max-w-xl'>
            <DialogHeader>
              <DialogTitle className='text-xl'>
                App key
              </DialogTitle>
            </DialogHeader>

            <div className='flex flex-col gap-2'>
              <span>
                Your app key:
              </span>
              <div className='flex flex-row items-center gap-2'>
                <Input readOnly value={payload} />
                <Button
                  size='icon'
                  variant='outline'
                  onClick={async () => {
                    await navigator.clipboard.writeText(payload ?? '');
                    setCopied(true);
                  }}
                >
                  {copied || <Copy />}
                  {copied && <Check />}
                </Button>
              </div>

              <div className='text-sm text-secondary-foreground flex flex-row justify-start items-center gap-2'>
                <Info size={12} className='inline' />
                <span>Store your key in a safe place. <b>You will not see this key again.</b></span>
              </div>

              <div className='text-sm text-secondary-foreground'>
                For details on how to utilize app keys, refer to our
                <Button
                  className='px-1'
                  variant='link'
                  nativeButton={false}
                  render={
                    <a target='_blank' href='https://doc.longhub.top'>document</a>
                  } />
              </div>
            </div>

            <DialogFooter className='mt-4'>
              <DialogClose
                render={
                  <Button>
                    Close
                  </Button>
                } />
            </DialogFooter>
          </DialogContent>
        )
      }}
    </Dialog>
  )
}