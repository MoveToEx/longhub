import api from '@/shared/lib/axios';

type GraphQLResponse<T> = {
  data?: T | null;
  errors?: { message: string }[];
};

export async function adminGraphQL<T>(
  query: string,
  variables: Record<string, unknown>,
  session: string,
) {
  const response = await api.post<GraphQLResponse<T>>(
    '/admin/graphql',
    { query, variables },
    {
      headers: { Authorization: 'Session ' + session },
    },
  );
  if (response.data.errors?.length) {
    throw new Error(response.data.errors.map((error) => error.message).join('; '));
  }
  if (!response.data.data) throw new Error('The admin query returned no data.');
  return response.data.data;
}
