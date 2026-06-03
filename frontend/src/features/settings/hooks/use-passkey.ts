import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";

type PasskeyResponse = {
  id: string,
  name: string,
  userId: number,
  aaguid: string,
  createdAt: string,
}[];

export default function usePasskey() {
  return useTaggedSWR<[], PasskeyResponse>({
    id: 'passkeys',
    args: [],
    fetcher: async () => {
      const response = await api.get('/user/webauthn');
      return response.data.data;
    },
    tags: ['settings', 'passkey', 'self'],
  });
}
