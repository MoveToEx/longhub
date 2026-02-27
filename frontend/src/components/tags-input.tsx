import { type RefCallBack } from 'react-hook-form';
import { Button } from "@/components/ui/button";
import { Edit, X } from "lucide-react";
import { Input } from '@/components/ui/input';
import { Autocomplete } from '@base-ui/react';
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card';
import { Spinner } from '@/components/ui/spinner';
import useTags from '@/hooks/server/use-tags';
import { useDebouncedState } from '@/hooks/use-debounced-state';

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
  const {
    debouncedState: staged,
    setState: setStaged,
    instantState: instant,
    isPending
  } = useDebouncedState('', 200);

  const { data: tags, isLoading } = useTags(staged);

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
        <Autocomplete.Root
          mode='both'
          items={!(isPending || isLoading) ? (tags?.tags ?? []) : []}>
          <label>
            <Autocomplete.Input placeholder='monochrome' render={
              <Input
                disabled={disabled}
                value={instant}
                onChange={e => setStaged(e.target.value)}
                onKeyDown={e => {
                  if (e.key === 'Enter') {
                    if (value.indexOf(instant) !== -1) return;
                    if (instant.indexOf(' ') !== -1 || instant.length === 0) return;

                    e.preventDefault();

                    onChange([...value, instant]);
                    setStaged('');
                    return false;
                  }
                }}
              />
            } />
          </label>

          <Autocomplete.Portal>
            <Autocomplete.Positioner className='z-60' sideOffset={4}>
              <Autocomplete.Popup className='data-open:animate-in data-closed:animate-out data-closed:fade-out-0 data-open:fade-in-0 data-closed:zoom-out-95 data-open:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 ring-foreground/10 bg-popover text-popover-foreground min-w-32 rounded-md p-1 shadow-md ring-1 duration-100 z-50 max-h-(--available-height) w-(--anchor-width) origin-(--transform-origin) overflow-x-hidden overflow-y-auto outline-none data-closed:overflow-hidden'>
                <Autocomplete.Status>
                  {(isLoading || isPending) && (
                    <Autocomplete.Item
                      disabled
                      className="focus:**:text-accent-foreground gap-2 rounded-sm px-2 py-1.5 text-sm [&_svg:not([class*='size-'])]:size-4 group/dropdown-menu-item relative flex cursor-default items-center outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-50 data-inset:pl-8 [&_svg]:pointer-events-none [&_svg]:shrink-0">
                      <Spinner /> Loading...
                    </Autocomplete.Item>
                  )}
                  {!isPending && !isLoading && tags && tags.tags.length === 0 && (
                    <Autocomplete.Item
                      disabled
                      className="hover:bg-accent hover:text-accent-foreground focus:**:text-accent-foreground gap-2 rounded-sm px-2 py-1.5 text-sm [&_svg:not([class*='size-'])]:size-4 group/dropdown-menu-item relative flex cursor-default items-center outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-50 data-inset:pl-8 [&_svg]:pointer-events-none [&_svg]:shrink-0">
                      <i>No tag found</i>
                    </Autocomplete.Item>
                  )}
                </Autocomplete.Status>

                <Autocomplete.List>
                  {tag => (
                    <Autocomplete.Item
                      key={tag.id}
                      value={tag.name}
                      onClick={() => {
                        if (value.indexOf(tag.name) !== -1) return;

                        setStaged('');
                        onChange([...value, tag.name]);
                      }}
                      className="hover:bg-accent hover:text-accent-foreground data-highlighted:bg-accent data-highlighted:text-accent-foreground focus:**:text-accent-foreground gap-2 rounded-sm px-2 py-1.5 text-sm [&_svg:not([class*='size-'])]:size-4 group/dropdown-menu-item relative flex cursor-default items-center outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-50 data-inset:pl-8 [&_svg]:pointer-events-none [&_svg]:shrink-0">
                      {tag.name}
                    </Autocomplete.Item>
                  )}
                </Autocomplete.List>
              </Autocomplete.Popup>
            </Autocomplete.Positioner>
          </Autocomplete.Portal>
        </Autocomplete.Root>
      </div>
    </div>
  )
}