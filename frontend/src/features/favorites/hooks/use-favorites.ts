import type { Image, Version } from "@/shared/lib/types";
import useTaggedSWR from "@/shared/lib/swr";

type ImagesResponse = {
  total: number,
  images: (Pick<Version, 'text' | 'rating'> & Pick<Image, 'id' | 'imageKey' | 'imageUrl' | 'userId'> & {
    shortcut: string | null,
    favoritedAt: string,
  })[]
}

export default function useFavorites(offset: number = 0, limit: number = 48) {
  return useTaggedSWR<[], ImagesResponse>({
    type: 'GET',
    url: '/favorite',
    query: {
      offset: offset.toString(),
      limit: limit.toString(),
    },
    tags: ['image', 'favorite', 'self']
  });
}