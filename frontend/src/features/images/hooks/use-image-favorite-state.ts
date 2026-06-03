import type { Favorite } from "@/shared/lib/types";
import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";
import useAuth from "@/features/auth/hooks/use-auth";

type FavoriteResponse = Favorite | null;

export default function useImageFavoriteState(id: number) {
  const { data } = useAuth();
  
  return useTaggedSWR<[number, boolean], FavoriteResponse>({
    id: 'image-favorite-state',
    args: [id, Boolean(data)],
    fetcher: async (id, enabled) => {
      if (!enabled) {
        return null;
      }

      const response = await api.get(`/favorite/${id}`);
      return response.data.data;
    },
    tags: ['image', 'self', 'favorite'],
  });
}
