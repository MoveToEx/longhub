import { useState } from 'react';
import { type RefCallBack } from 'react-hook-form';
import { Button } from "@/shared/components/ui/button";
import { Edit, X } from "lucide-react";
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/shared/components/ui/hover-card';
import TagInput from '@/features/tags/components/tag-input';

type TagsInputProps = {
  value: string[],
  onChange: (val: string[]) => void,
  disabled?: boolean;
  name: string,
  ref: RefCallBack
}

export default function TagsInput({
  value,
  onChange,
  disabled = false
}: TagsInputProps) {
  const [staged, setStaged] = useState('');

  return (
    <div>
      <div className='w-full mb-1 flex flex-row justify-start items-center flex-wrap'>
        {value.map(val => (
          <HoverCard key={val}>
            <HoverCardTrigger delay={100} closeDelay={100}>
              <span key={val} className='rounded-sm px-2 py-1 mx-1 my-1 border border-gray-400 flex items-center'>
                {val}
              </span>
            </HoverCardTrigger>

            <HoverCardContent className='flex flex-row gap-2 w-fit'>
              <Button
                variant='ghost'
                onClick={() => {
                  onChange(value.filter(it => it !== val));
                  setStaged(val);
                }}>
                <Edit /> Edit
              </Button>
              <Button
                variant='ghost'
                onClick={() => {
                  onChange(value.filter(it => it !== val))
                }}>
                <X /> Delete
              </Button>
            </HoverCardContent>
          </HoverCard>
        ))}
      </div>

      <div>
        <TagInput
          disabled={disabled}
          value={staged}
          onValueChange={setStaged}
          onTagSelect={tag => {
            if (value.indexOf(tag) !== -1) return;
            onChange([...value, tag]);
            setStaged('');
          }}
        />
      </div>
    </div>
  )
}
