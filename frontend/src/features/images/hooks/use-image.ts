import type { Image, Tag, UserIdentifier, Version } from "@/shared/lib/types";
import useTaggedSWR from "@/shared/lib/swr";

type ImageResponse = Image & Pick<Version, 'text' | 'rating' | 'version'> & {
  userIdentifier: UserIdentifier,
  tags: Tag[]
}

export default function useImage(id: number) {
  return useTaggedSWR<[], ImageResponse>({
    type: 'GET',
    url: `/image/${id}`,
    tags: ['image']
  });
}