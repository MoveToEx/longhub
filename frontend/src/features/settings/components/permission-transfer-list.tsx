import { Button } from "@/shared/components/ui/button";
import { useMemo, useState } from "react";
import { ScrollArea, ScrollBar } from "@/shared/components/ui/scroll-area";
import { Permission } from "@/shared/lib/const";
import { cn } from "@/shared/lib/utils";

type Props = {
  mask: number,
  value: number,
  onValueChange: (val: number) => void,
  disabled?: boolean,
}

export default function PermissionTransferList({ mask, value, onValueChange, disabled = false }: Props) {
  const _toArray = (p: number) => {
    return Object.keys(Permission).map(Number).filter(bit => (p & bit) !== 0);
  }

  const current = useMemo(() => _toArray(value), [value]);
  const rest = useMemo(() => _toArray(mask & (~value)), [mask, value]);

  const [left, setLeft] = useState([] as number[]);
  const [right, setRight] = useState([] as number[]);
  const selectedAvailable = left.filter(bit => rest.includes(bit));
  const selectedGranted = right.filter(bit => current.includes(bit));

  return (
    <div className='h-48 flex flex-row gap-4 sm:gap-8'>
      <div role='group' aria-label='Available permissions' className='min-w-0 flex-1 flex flex-col items-center'>
        <span className='text-md mb-2'>
          Permission
        </span>
        <ScrollArea className='flex-1 w-full h-32 border border-b-accent rounded-md p-1'>
          {rest.map(it => (
            <button
              key={it}
              type='button'
              disabled={disabled}
              aria-pressed={selectedAvailable.includes(it)}
              className={cn('w-full text-left hover:bg-accent px-2 py-0.5 rounded-sm disabled:opacity-50', selectedAvailable.includes(it) ? 'bg-accent' : '')}
              onClick={() => {
                if (left.indexOf(it) === -1) {
                  setLeft(prev => [...prev, it]);
                }
                else {
                  setLeft(prev => prev.filter(val => val !== it));
                }
              }}>
              {Permission[it]}
            </button>
          ))}
          <ScrollBar />
        </ScrollArea>
      </div>

      <div className='h-full flex flex-col gap-2 justify-center'>
        <Button
          type='button'
          variant='outline'
          aria-label='Grant selected permissions'
          disabled={disabled || selectedAvailable.length === 0}
          onClick={() => {
            onValueChange(value | selectedAvailable.reduce((prev, cur) => prev | cur, 0));
            setLeft([]);
          }}>
          &gt;&gt;
        </Button>
        <Button
          type='button'
          variant='outline'
          aria-label='Revoke selected permissions'
          disabled={disabled || selectedGranted.length === 0}
          onClick={() => {
            onValueChange(value & ~selectedGranted.reduce((prev, cur) => prev | cur, 0));
            setRight([]);
          }}
        >
          &lt;&lt;
        </Button>
      </div>

      <div role='group' aria-label='Granted permissions' className='min-w-0 flex-1 flex flex-col items-center'>
        <span className='text-md mb-2'>
          Granted
        </span>
        <ScrollArea className='flex-1 w-full h-32 border border-accent rounded-md p-1'>
          {current.map(it => (
            <button
              key={it}
              type='button'
              disabled={disabled}
              aria-pressed={selectedGranted.includes(it)}
              className={cn('w-full text-left hover:bg-accent px-2 py-0.5 rounded-sm disabled:opacity-50', selectedGranted.includes(it) ? 'bg-accent' : '')}
              onClick={() => {
                if (right.indexOf(it) === -1) {
                  setRight(prev => [...prev, it]);
                }
                else {
                  setRight(prev => prev.filter(val => val !== it));
                }
              }}>
              {Permission[it]}
            </button>
          ))}
          <ScrollBar />
        </ScrollArea>
      </div>
    </div>
  )
}
