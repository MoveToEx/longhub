import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";

type ContributionResponse = {
  day: string,
  imageCount: number,
  versionCount: number
}[]

export default function useUserContribution(userId: number) {
  return useTaggedSWR<[number], ContributionResponse>({
    id: 'user-contribution',
    args: [userId],
    fetcher: async (userId) => {
      const response = await api.get(`/user/${userId}/contribution`);
      return response.data.data;
    },
    tags: ['user', 'contribution'],
  });
}
