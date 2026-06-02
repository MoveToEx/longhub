import type { UserIdentifier } from "@/shared/lib/types";
import useTaggedSWR from "@/shared/lib/swr";

type UserResponse = UserIdentifier & {
  versions: number,
  images: number,
};

export default function useUser(id: number) {
  return useTaggedSWR<[], UserResponse>({
    type: 'GET',
    url: `/user/${id}`,
    tags: ['user']
  });
}