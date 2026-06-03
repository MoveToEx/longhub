import type { Image, Version } from "@/shared/lib/types";
import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";

type ImagesResponse = {
  total: number,
  images: (Pick<Version, 'text' | 'rating'> & Pick<Image, 'id' | 'imageKey' | 'imageUrl' | 'userId'> & {
    shortcut: string | null,
    favoritedAt: string,
  })[]
}

export default function useFavorites(offset: number = 0, limit: number = 48) {
  return useTaggedSWR<[number, number], ImagesResponse>({
    id: 'favorites',
    args: [offset, limit],
    fetcher: async (offset, limit) => {
      const response = await api.get('/favorite', {
        params: {
          offset: offset.toString(),
          limit: limit.toString(),
        },
      });
      return response.data.data;
    },
    tags: ['image', 'favorite', 'self'],
  });
}
