import { useState, useRef, forwardRef, type ForwardedRef } from 'react';
import wcmatch from 'wildcard-match';
import _ from 'lodash';
import { Upload } from 'lucide-react';
import { Button } from './ui/button';
import { cn } from '@/lib/utils';

const DragDrop = forwardRef(function ({
  className = '',
  multiple = false,
  accept = '*/*',
  onChange,
  ...rest
}: {
  className?: string,
  multiple?: boolean,
  accept?: string | string[],
  onChange: (files: Blob[]) => void
}, ref: ForwardedRef<HTMLDivElement>) {
  const [dragging, setDragging] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const match = wcmatch(accept);

  return (
    <div
      ref={ref}
      {...rest}
      className={
        cn(
          'flex flex-col items-center justify-center rounded border-2 border-dashed border-gray-400 dark:border-gray-700 w-full min-h-48', 
          dragging ? '**:pointer-events-none bg-gray-200 dark:bg-gray-800 border-blue-400 dark:border-blue-900' : '',
          className,
        )
      }
      onDragOver={(e) => {
        e.preventDefault();
        e.stopPropagation();
      }}
      onDragEnter={(e) => {
        e.preventDefault();
        e.stopPropagation();
        if (!dragging) setDragging(true);
      }}
      onDragLeave={(e) => {
        e.preventDefault();
        e.stopPropagation();
        if (dragging) setDragging(false);
      }}
      onDrop={(e) => {
        e.preventDefault();
        e.stopPropagation();
        setDragging(false);

        const items = [...e.dataTransfer.items].filter(item => {
          return item.kind === 'file' && match(item.type);
        });

        const result = _.compact(items.map(item => item.getAsFile()));

        if (result.length > 0) {
          onChange(result);
        }
      }}>
      <input
        ref={inputRef}
        className='hidden'
        type='file'
        accept={typeof accept === 'string' ? accept : accept.join(',')}
        onChange={(e) => {
          if (e.target.files) onChange([...e.target.files]);
        }}
        multiple={multiple} />
      <Upload size={48} />
      <span className='text-md text-center'>
        Drop files here,&nbsp;
        <Button
          variant='link'
          className='p-0'
          onClick={() => {
            if (inputRef.current) {
              inputRef.current.click();
            }
          }}>
          browse
        </Button>
        , or&nbsp;
        <Button
          variant='link'
          className='p-0'
          onClick={async () => {
            const items = await navigator.clipboard.read();
            const result = [];
            for (const item of items) {
              for (const type of item.types) {
                if (match(type)) {
                  result.push(await item.getType(type));
                  break;
                }
              }
            }
            onChange(result);
          }}>
          paste
        </Button>
      </span>
    </div>
  );
});

export default DragDrop;