import type { Image, Version } from "@/shared/lib/types";
import useTaggedSWR from "@/shared/lib/swr";

type ImagesResponse = {
  total: number,
  images: (Pick<Version, 'text' | 'rating'> & Image)[]
}

export default function useImages(offset: number = 0, limit: number = 48) {
  return useTaggedSWR<[], ImagesResponse>({
    type: 'GET',
    url: '/image',
    query: {
      offset: offset.toString(),
      limit: limit.toString(),
    },
    tags: ['image']
  });
}