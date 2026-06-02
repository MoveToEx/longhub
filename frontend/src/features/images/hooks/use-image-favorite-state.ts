import type { Favorite } from "@/shared/lib/types";
import useTaggedSWR from "@/shared/lib/swr";
import useAuth from "@/features/auth/hooks/use-auth";

type FavoriteResponse = Favorite | null;

export default function useImageFavoriteState(id: number) {
  const { data } = useAuth();
  
  return useTaggedSWR<[], FavoriteResponse>({
    type: 'GET',
    url: `/favorite/${id}`,
    tags: ['image', 'self', 'favorite'],
    disabled: !data,
  });
}
