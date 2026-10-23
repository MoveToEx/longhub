export type SortDirection = 'ASC' | 'DESC';
export type ImageStatus = 'ACTIVE' | 'DELETED' | 'ALL';

export type AdminQueryInput = {
  offset: number;
  limit: number;
  search: string;
  sortBy: string;
  direction: SortDirection;
  status?: ImageStatus;
};

export type AdminPage<T> = { items: T[]; total: number };

export type AdminImage = {
  id: string;
  imageUrl: string;
  imageKey: string;
  uploader: { id: string; username: string };
  createdAt: string;
  updatedAt: string;
  deletedAt: string | null;
  currentVersion: {
    id: string;
    version: number;
    text: string;
    rating: 'none' | 'moderate' | 'violent';
    tags: string[];
  } | null;
};

export type AdminUser = {
  id: string;
  username: string;
  email: string;
  permission: number;
  createdAt: string;
  activeImageCount: number;
  versionCount: number;
};

export const IMAGE_QUERY = `query AdminImages($input: ImageQueryInput) {
  images(input: $input) {
    total
    items {
      id imageUrl imageKey createdAt updatedAt deletedAt
      uploader { id username }
      currentVersion { id version text rating tags }
    }
  }
}`;

export const USER_QUERY = `query AdminUsers($input: UserQueryInput) {
  users(input: $input) {
    total
    items { id username email permission createdAt activeImageCount versionCount }
  }
}`;
