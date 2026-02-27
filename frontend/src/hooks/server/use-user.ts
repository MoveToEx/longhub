import type { UserIdentifier } from "@/lib/types";
import useTaggedSWR from "@/lib/swr";

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