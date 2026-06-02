import type { AppKey } from "@/shared/lib/types";
import useTaggedSWR from "@/shared/lib/swr";

type AppKeyResponse = AppKey[];

export default function useAppKeys() {
  return useTaggedSWR<[], AppKeyResponse>({
    type: 'GET',
    url: '/user/appkey',
    tags: ['appkey', 'self']
  });
}