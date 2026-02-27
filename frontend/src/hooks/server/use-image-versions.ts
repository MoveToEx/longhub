import type { UserIdentifier, Version } from "@/lib/types";
import useTaggedSWR from "@/lib/swr";

type VersionResponse = (Version & {
  userIdentifier: UserIdentifier,
  tags: string[]
})[]

export default function useImageVersions(id: number) {
  return useTaggedSWR<[], VersionResponse>({
    type: 'GET',
    url: `/image/${id}/version`,
    tags: ['image']
  });
}