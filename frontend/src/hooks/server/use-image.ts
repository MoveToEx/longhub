import type { Image, Tag, UserIdentifier, Version } from "@/lib/types";
import useTaggedSWR from "@/lib/swr";

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