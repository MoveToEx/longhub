import useTaggedSWR from "@/lib/swr";

type PasskeyResponse = {
  id: string,
  name: string,
  userId: number,
  aaguid: string,
  createdAt: string,
}[];

export default function usePasskey() {
  return useTaggedSWR<[], PasskeyResponse>({
    type: 'GET',
    url: '/user/webauthn',
    tags: ['settings', 'passkey', 'self']
  });
}