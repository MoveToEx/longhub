import { Button } from "@/shared/components/ui/button";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/shared/components/ui/hover-card";
import { Pagination, PaginationContent, PaginationItem, PaginationLink, PaginationNext, PaginationPrevious } from "@/shared/components/ui/pagination";
import { Spinner } from "@/shared/components/ui/spinner";
import useFavorites from "@/features/favorites/hooks/use-favorites"
import { copyImage } from "@/shared/lib/utils";
import { Bookmark, Copy } from "lucide-react";
import { useMemo, useState } from "react";
import { useSearchParams } from "react-router";
import { toast } from "sonner";
import { RequiresLogin } from "@/shared/components/utils";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/shared/components/ui/empty";
import LikeButton from "@/features/images/components/like-button";

function CopyButton({
  url
}: {
  url: string,
}) {
  const [loading, setLoading] = useState(false);

  return (
    <Button
      variant='outline'
      disabled={loading}
      onClick={async () => {
        setLoading(true);
        try {
          await copyImage(url);
          toast.success('Copied to clipboard');
        }
        catch (e) {
          if (e instanceof Error) {
            toast.error(e.message);
          }
        }
        finally {
          setLoading(false);
        }
      }}>
      {loading && <Spinner />}
      {loading || <Copy />}
      Copy
    </Button>
  )
}

function Image({ id, imageUrl, shortcut, date }: {
  id: number,
  imageUrl: string,
  shortcut: string | null,
  date: Date,
}) {
  return (
    <HoverCard>
      <HoverCardTrigger delay={500} closeDelay={200} render={
        <img src={imageUrl} className='w-full h-48 object-contain object-center' crossOrigin="anonymous" />
      } />
      <HoverCardContent className='flex flex-col items-center gap-4 w-72'>
        <div className='w-full flex flex-col items-center gap-2'>
          <div>
            Shortcut: {shortcut === null ? <i className='text-muted-foreground'>null</i> : (
              <span className='border-accent-foreground border bg-accent font-mono py-0.5 px-1 rounded-sm'>
                {shortcut}
              </span>
            )}
          </div>
          <div>
            Favorited on {date.toDateString()}
          </div>
        </div>
        <div className='flex flex-row justify-evenly gap-4 mx-4'>
          <LikeButton id={id} />
          <CopyButton url={imageUrl} />
        </div>
      </HoverCardContent>
    </HoverCard>
  )
}

export default function FavoritesPage() {
  const [searchParams, setSearchParams] = useSearchParams({
    p: "1"
  });
  const page = useMemo(() => Number(searchParams.get('p') ?? 1), [searchParams]);
  const { data, isLoading } = useFavorites();

  return (
    <div className='w-full h-full'>
      <RequiresLogin />

      {isLoading && (
        <div className='w-full h-32 flex flex-row items-center justify-center gap-2'>
          <Spinner />
          Loading
        </div>
      )}

      {data?.total === 0 && (
        <Empty className='h-64'>
          <EmptyHeader>
            <EmptyMedia variant='icon'>
              <Bookmark />
            </EmptyMedia>
            <EmptyTitle>
              No favorites yet
            </EmptyTitle>
            <EmptyDescription>
              You haven't favorited any image.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
      <div className='grid grid-cols-6 gap-2'>
        {data?.images.map(it => (
          <div key={it.id}>
            <Image
              id={it.id}
              imageUrl={it.imageUrl}
              shortcut={it.shortcut}
              date={new Date(it.favoritedAt)}
            />
          </div>
        ))}
      </div>

      {!!data?.total && (
        <Pagination className='mt-4 mb-2'>
          <PaginationContent>
            <PaginationItem>
              <PaginationPrevious
                onClick={() => {
                  if (page === 1) return;

                  setSearchParams({
                    p: (page - 1).toString()
                  });
                }} />
            </PaginationItem>

            {page > 2 && (
              <PaginationItem>
                <PaginationLink onClick={() => {
                  setSearchParams({
                    p: (page - 1).toString()
                  })
                }}>{page - 1}</PaginationLink>
              </PaginationItem>
            )}

            <PaginationItem>
              <PaginationLink isActive>{page}</PaginationLink>
            </PaginationItem>

            <PaginationItem>
              <PaginationNext
                onClick={() => {
                  if (page * 48 >= (data?.total ?? 0)) {
                    return;
                  }

                  setSearchParams({
                    p: String(page + 1)
                  })
                }} />
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      )}

    </div>
  )
}
