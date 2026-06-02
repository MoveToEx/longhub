import useTaggedSWR from "@/shared/lib/swr";

type ContributionResponse = {
  day: string,
  imageCount: number,
  versionCount: number
}[]

export default function useUserContribution(userId: number) {
  return useTaggedSWR<[], ContributionResponse>({
    type: 'GET',
    url: `/user/${userId}/contribution`,
    tags: ['user', 'contribution']
  });
}