import { Button } from "@/components/ui/button";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Pagination, PaginationContent, PaginationItem, PaginationLink, PaginationNext, PaginationPrevious } from "@/components/ui/pagination";
import { Spinner } from "@/components/ui/spinner";
import useFavorites from "@/hooks/server/use-favorites"
import useImageFavoriteState from "@/hooks/server/use-image-favorite-state";
import useAuth from "@/hooks/server/use-auth";
import api from "@/lib/axios";
import { Dialog as BaseDialog } from "@base-ui/react";
import { mutate } from "@/lib/swr";
import { copyImage } from "@/lib/utils";
import { Bookmark, Copy, Edit, Heart } from "lucide-react";
import { useMemo, useState } from "react";
import { useSearchParams } from "react-router";
import { toast } from "sonner";
import EditShortcutDialog, { type Payload as ShortcutPayload } from "@/components/dialogs/edit-shortcut";
import { RequiresLogin } from "@/components/utils";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";

function LikeButton({
  id
}: {
  id: number
}) {
  const { data: user } = useAuth();
  const { data, isValidating } = useImageFavoriteState(id);

  return (
    <Button
      onClick={async () => {
        await api.delete(`/favorite/${id}`);
        await mutate('favorite');
        toast.success('Removed from favorites');
      }}
      variant='outline'
      disabled={isValidating || !user}>
      {isValidating && <Spinner />}
      {!isValidating && <Heart fill={data ? '#f84a63' : '#ffffff'} />}
      Like
    </Button>
  )
}

function EditButton({ id, onClick }: { id: number, onClick: () => void }) {
  const { isValidating } = useImageFavoriteState(id);
  return (
    <Button
      onClick={() => onClick()}
      variant='outline'>
      {isValidating && <Spinner />}
      {!isValidating && <Edit />}
      Edit
    </Button>
  )
}

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

function Image({ id, imageUrl, shortcut, date, onEdit }: {
  id: number,
  imageUrl: string,
  shortcut: string | null,
  date: Date,
  onEdit: () => void,
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
          <EditButton id={id} onClick={() => onEdit()} />
          <CopyButton url={imageUrl} />
        </div>
      </HoverCardContent>
    </HoverCard>
  )
}

const editHandle = BaseDialog.createHandle<ShortcutPayload>();

export default function FavoritesPage() {
  const [searchParams, setSearchParams] = useSearchParams({
    p: "1"
  });
  const page = useMemo(() => Number(searchParams.get('p') ?? 1), [searchParams]);
  const { data, isLoading } = useFavorites();

  return (
    <div className='w-full h-full'>
      <RequiresLogin />
      <EditShortcutDialog handle={editHandle} />

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
              onEdit={() => {
                editHandle.openWithPayload({ ...it, shortcut: it.shortcut ?? '' });
              }}
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