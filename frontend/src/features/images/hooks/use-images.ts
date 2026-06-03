import type { Image, Version } from "@/shared/lib/types";
import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";

type ImagesResponse = {
  total: number,
  images: (Pick<Version, 'text' | 'rating'> & Image)[]
}

export default function useImages(offset: number = 0, limit: number = 48) {
  return useTaggedSWR<[number, number], ImagesResponse>({
    id: 'images',
    args: [offset, limit],
    fetcher: async (offset, limit) => {
      const response = await api.get('/image', {
        params: {
          offset: offset.toString(),
          limit: limit.toString(),
        },
      });
      return response.data.data;
    },
    tags: ['image'],
  });
}
