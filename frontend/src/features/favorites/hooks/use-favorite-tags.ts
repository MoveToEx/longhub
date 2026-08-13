import type { Image, Tag } from "@/shared/lib/types";
import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";

type FavoriteTagsResponse = (Tag & {
  count: number,
  images: Pick<Image, 'id' | 'imageUrl' | 'imageKey'>[],
})[]

export default function useFavoriteTags() {
  return useTaggedSWR<[], FavoriteTagsResponse>({
    id: 'favorite-tags',
    args: [],
    fetcher: async () => {
      const response = await api.get('/recommend');
      return response.data.data;
    },
    tags: ['tag', 'favorite', 'self'],
    immutable: true,
    config: {
      revalidateIfStale: false,
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
    },
  });
}
