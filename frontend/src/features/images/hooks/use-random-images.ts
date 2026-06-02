import type { Image } from "@/shared/lib/types";
import useTaggedSWR from "@/shared/lib/swr";

type RandomImagesResponse = Pick<Image, 'id' | 'imageUrl' | 'imageKey'>[]

export default function useRandomImages(tag: string) {
  return useTaggedSWR<[], RandomImagesResponse>({
    type: 'GET',
    url: `/tag/random/${tag}`,
    query: {
      limit: '12'
    },
    tags: ['tag', 'favorite', 'self'],
    immutable: true
  });
}