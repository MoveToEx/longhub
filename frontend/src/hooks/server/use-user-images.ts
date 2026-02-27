import useTaggedSWR from "@/lib/swr";
import type { Image, Version } from "@/lib/types";

type UserImagesResponse = {
  total: number,
  images: (Pick<Version, 'text' | 'rating'> & Pick<Image, 'id' | 'imageUrl' | 'currentVersionId'>)[]
}

export default function useUserImages(userId: number) {
  return useTaggedSWR<[], UserImagesResponse>({
    type: 'GET',
    url: `/user/${userId}/image`,
    tags: ['user', 'image']
  });
}