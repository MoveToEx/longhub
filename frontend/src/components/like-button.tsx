import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import useImageFavoriteState from "@/hooks/server/use-image-favorite-state";
import useAuth from "@/hooks/server/use-auth";
import api from "@/lib/axios";
import { Heart } from "lucide-react";
import { toast } from "sonner";

export default function LikeButton({
  id,
  customizeAction,
}: {
  id: number,
  customizeAction: () => void
}) {
  const { data: user } = useAuth();
  const { data, isValidating, mutate } = useImageFavoriteState(id);

  return (
    <Button
      onClick={async () => {
        if (data) {
          await api.delete(`/favorite/${id}`);
        }
        else {
          await api.post('/favorite', {
            imageId: id
          });
        }
        await mutate();
        if (data) {
          toast.success('Removed from favorites');
        }
        else {
          toast.success('Added to favorites', {
            action: {
              label: 'Customize',
              onClick: () => customizeAction()
            }
          })
        }
      }}
      variant='outline'
      disabled={isValidating || !user}>
      {isValidating && <Spinner />}
      {!isValidating && <Heart fill={data ? '#f84a63' : '#ffffff'} />}
      Like
    </Button>
  )
}