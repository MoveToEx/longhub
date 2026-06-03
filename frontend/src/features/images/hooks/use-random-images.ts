import type { Image } from "@/shared/lib/types";
import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";

type RandomImagesResponse = Pick<Image, 'id' | 'imageUrl' | 'imageKey'>[]

export default function useRandomImages(tag: string) {
  return useTaggedSWR<[string], RandomImagesResponse>({
    id: 'random-images',
    args: [tag],
    fetcher: async (tag) => {
      const response = await api.get(`/tag/random/${tag}`, {
        params: {
          limit: '12',
        },
      });
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
