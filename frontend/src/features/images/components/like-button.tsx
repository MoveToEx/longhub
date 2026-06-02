import { Button } from "@/shared/components/ui/button";
import { Spinner } from "@/shared/components/ui/spinner";
import useImageFavoriteState from "@/features/images/hooks/use-image-favorite-state";
import useAuth from "@/features/auth/hooks/use-auth";
import api from "@/shared/lib/axios";
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