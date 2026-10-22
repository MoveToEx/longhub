import { Button } from "@/shared/components/ui/button";
import { Spinner } from "@/shared/components/ui/spinner";
import useImageFavoriteState from "@/features/images/hooks/use-image-favorite-state";
import useAuth from "@/features/auth/hooks/use-auth";
import api from "@/shared/lib/axios";
import { Heart } from "lucide-react";
import { toast } from "sonner";
import { useAppDispatch } from "@/shared/hooks/use-redux";
import { openEditShortcutDialog } from "@/features/images/state/edit-shortcut-dialog-slice";
import { formatError } from "@/shared/lib/utils";
import { useState } from "react";

export default function LikeButton({
  id,
}: {
  id: number,
}) {
  const dispatch = useAppDispatch();
  const { data: user } = useAuth();
  const { data, isLoading, mutate } = useImageFavoriteState(id);
  const [isFavoriting, setIsFavoriting] = useState(false);
  const loading = isLoading || isFavoriting;

  return (
    <Button
      onClick={async () => {
        if (data) {
          dispatch(openEditShortcutDialog({ id, shortcut: data.shortcut ?? '' }));
          return;
        }

        setIsFavoriting(true);
        try {
          await api.post('/favorite', {
            imageId: id
          });
          await mutate();
          toast.success('Added to favorites', {
            action: {
              label: 'Customize',
              onClick: () => dispatch(openEditShortcutDialog({ id, shortcut: '' }))
            }
          })
        }
        catch (e) {
          toast.error(formatError(e));
        }
        finally {
          setIsFavoriting(false);
        }
      }}
      variant='outline'
      aria-busy={loading}
      disabled={loading || !user}>
      {loading && <Spinner />}
      {!loading && <Heart fill={data ? '#f84a63' : '#ffffff'} />}
      {data && <span>Customize</span>}
      {!!data || <span>Like</span>}
    </Button>
  )
}
