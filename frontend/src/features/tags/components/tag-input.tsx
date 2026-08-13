import { Autocomplete } from '@base-ui/react';
import { Input } from '@/shared/components/ui/input';
import { Spinner } from '@/shared/components/ui/spinner';
import useTags from '@/features/tags/hooks/use-tags';

type TagInputProps = {
  value: string,
  onValueChange: (value: string) => void,
  onTagSelect: (value: string) => void,
  disabled?: boolean,
  placeholder?: string,
  className?: string,
  required?: boolean,
}

export default function TagInput({
  value,
  onValueChange,
  onTagSelect,
  disabled = false,
  placeholder = 'monochrome',
  className,
  required = false,
}: TagInputProps) {
  const { data: tags, isLoading } = useTags(value);

  const selectTag = (tag: string) => {
    const normalized = tag.trim();
    if (normalized.length === 0 || normalized.indexOf(' ') !== -1) return;
    onTagSelect(normalized);
  };

  return (
    <div className={className}>
      <Autocomplete.Root
        mode='both'
        items={!isLoading ? (tags?.tags ?? []) : []}>
        <label>
          <Autocomplete.Input placeholder={placeholder} render={
            <Input
              required={required}
              disabled={disabled}
              value={value}
              onChange={event => onValueChange(event.target.value)}
              onKeyDown={event => {
                if (event.key !== 'Enter') return;
                event.preventDefault();
                selectTag(value);
              }}
            />
          } />
        </label>

        <Autocomplete.Portal>
          <Autocomplete.Positioner className='z-60' sideOffset={4}>
          <Autocomplete.Popup className='data-open:animate-in data-closed:animate-out data-closed:fade-out-0 data-open:fade-in-0 data-closed:zoom-out-95 data-open:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 ring-foreground/10 bg-popover text-popover-foreground min-w-32 rounded-md p-1 shadow-md ring-1 duration-100 z-50 max-h-(--available-height) w-(--anchor-width) origin-(--transform-origin) overflow-x-hidden overflow-y-auto outline-none data-closed:overflow-hidden'>
            <Autocomplete.Status>
              {isLoading && (
                <Autocomplete.Item
                  disabled
                  className="focus:**:text-accent-foreground gap-2 rounded-sm px-2 py-1.5 text-sm [&_svg:not([class*='size-'])]:size-4 group/dropdown-menu-item relative flex cursor-default items-center outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-50 data-inset:pl-8 [&_svg]:pointer-events-none [&_svg]:shrink-0">
                  <Spinner /> Loading...
                </Autocomplete.Item>
              )}
              {!isLoading && tags && tags.tags.length === 0 && (
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
                  onClick={() => selectTag(tag.name)}
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
  );
}
