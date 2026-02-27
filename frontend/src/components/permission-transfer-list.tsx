import { Button } from "@/components/ui/button";
import { useMemo, useState } from "react";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { Permission } from "@/lib/const";
import { cn } from "@/lib/utils";

type Props = {
  mask: number,
  value: number,
  onValueChange: (val: number) => void,
}

export default function PermissionTransferList({ mask, value, onValueChange }: Props) {
  const _toArray = (p: number) => {
    const result: number[] = [];
    while (p !== 0) {
      const low = p & (-p);
      result.push(low);
      p ^= low;
    }
    return result;
  }

  const current = useMemo(() => _toArray(value), [value]);
  const rest = useMemo(() => _toArray(mask & (~value)), [mask, value]);

  const [left, setLeft] = useState([] as number[]);
  const [right, setRight] = useState([] as number[]);

  return (
    <div className='h-48 flex flex-row gap-8'>
      <div className='flex-1 flex flex-col items-center'>
        <span className='text-md mb-2'>
          Permission
        </span>
        <ScrollArea className='flex-1 w-full h-32 border border-b-accent rounded-md p-1'>
          {rest.map(it => (
            <div
              key={it}
              className={cn('hover:bg-accent px-2 py-0.5 rounded-sm', left.indexOf(it) === -1 ? '' : 'bg-accent')}
              onClick={() => {
                if (left.indexOf(it) === -1) {
                  setLeft(prev => [...prev, it]);
                }
                else {
                  setLeft(prev => prev.filter(val => val !== it));
                }
              }}>
              {Permission[it]}
            </div>
          ))}
          <ScrollBar />
        </ScrollArea>
      </div>

      <div className='h-full flex flex-col gap-2 justify-center'>
        <Button
          variant='outline'
          disabled={left.length === 0}
          onClick={() => {
            onValueChange(value | left.reduce((prev, cur) => prev | cur));
            setLeft([]);
          }}>
          &gt;&gt;
        </Button>
        <Button
          variant='outline'
          disabled={right.length === 0}
          onClick={() => {
            onValueChange(value & ~right.reduce((prev, cur) => prev | cur));
            setRight([]);
          }}
        >
          &lt;&lt;
        </Button>
      </div>

      <div className='flex-1 flex flex-col items-center'>
        <span className='text-md mb-2'>
          Granted
        </span>
        <ScrollArea className='flex-1 w-full h-32 border border-accent rounded-md p-1'>
          {current.map(it => (
            <div
              key={it}
              className={cn('hover:bg-accent px-2 py-0.5 rounded-sm', right.indexOf(it) === -1 ? '' : 'bg-accent')}
              onClick={() => {
                if (right.indexOf(it) === -1) {
                  setRight(prev => [...prev, it]);
                }
                else {
                  setRight(prev => prev.filter(val => val !== it));
                }
              }}>
              {Permission[it]}
            </div>
          ))}
          <ScrollBar />
        </ScrollArea>
      </div>
    </div>
  )
}