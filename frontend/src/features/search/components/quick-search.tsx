import { Dialog, DialogContent, DialogTrigger } from "@/shared/components/ui/dialog";
import { Dialog as BaseDialog } from '@base-ui/react';
import { Button } from "@base-ui/react";
import { Bookmark, Search } from "lucide-react";
import { useDebouncedState } from "@/shared/hooks/use-debounced-state";
import useQuickSearch from "@/features/search/hooks/use-quick-search";
import type { QuickSearchImage } from "@/features/search/hooks/use-quick-search";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/shared/components/ui/input-group";
import { ScrollArea } from "@/shared/components/ui/scroll-area";
import { Spinner } from "@/shared/components/ui/spinner";

type QuickSearchItem = QuickSearchImage & {
  favorited: boolean;
};

const handle = BaseDialog.createHandle();

export default function QuickSearch() {
  const { debouncedState, setState, instantState, isPending } = useDebouncedState('', 500);
  const { data, isLoading } = useQuickSearch(debouncedState.split(' '));
  const results: QuickSearchItem[] = data
    ? [
      ...data.favorited.map(it => ({ ...it, favorited: true })),
      ...data.filtered.map(it => ({ ...it, favorited: false })),
    ]
    : [];

  return (
    <Dialog handle={handle} onOpenChangeComplete={open => {
      if (open === false) {
        setState('');
      }
    }}>
      <DialogTrigger render={
        <Button className='xs:hidden md:flex w-12 md:w-48 rounded-md border border-gray-300 text-sm py-2 px-4 flex flex-row items-center justify-start gap-2 cursor-text text-muted-foreground hover:text-foreground'>
          <Search className='inline' size={16} />
          <span className='md:block hidden'>
            Quick search...
          </span>
        </Button>
      } />

      <DialogContent className='border-4 border-gray-100 sm:max-w-xl p-2 gap-2' showCloseButton={false}>
        <InputGroup>
          <InputGroupInput
            placeholder='Search anything...'
            value={instantState}
            onChange={e => setState(e.target.value)} />
          <InputGroupAddon>
            <Search />
          </InputGroupAddon>
        </InputGroup>

        <div className='h-96 flex flex-col'>
          {isLoading || isPending && (
            <div className='w-full h-full flex-1 flex items-center justify-center flex-row gap-2'>
              <Spinner className='text-gray-600' /> Loading
            </div>
          )}

          {instantState && !isLoading && !isPending && results.length === 0 && (
            <div className='w-full h-full flex-1 flex items-center justify-center flex-row gap-2'>
              <span className='italic text-muted-foreground'>
                No result found
              </span>
            </div>
          )}

          {!isLoading && !isPending && results.length !== 0 && (
            <ScrollArea className='w-full h-full'>
              {results.map(it => (
                <a href={`/#/image/${it.id}`} onClick={() => handle.close()}>
                  <div key={`${it.favorited ? 'favorited' : 'filtered'}-${it.id}`} className='w-full h-24 p-2 rounded-md hover:bg-gray-100 flex flex-row'>
                    <div className='h-full shrink-0 w-24'>
                      <img className='h-full w-24 object-contain' src={it.imageUrl} crossOrigin="anonymous" />
                    </div>
                    <div className='p-2 flex flex-col h-full'>
                      <span className='text-wrap truncate text-ellipsis max-w-full'>
                        {it.text}
                      </span>
                    </div>

                    <div className='flex-1' />
                    {it.favorited && (
                      <div className='h-full flex flex-row items-center gap-2'>
                        <span className='bg-gray-100 border border-gray-250 rounded-md px-2 text-muted-foreground p-1'>
                          <Bookmark size={12} />
                        </span>
                      </div>
                    )}
                  </div>
                </a>

              ))}
            </ScrollArea>
          )}
        </div>

      </DialogContent>
    </Dialog>
  )
}
