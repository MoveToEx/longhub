export type SortDirection = "ASC" | "DESC";
export type ImageStatus = "ACTIVE" | "DELETED" | "ALL";
export type DeletionStatus = "ACTIVE" | "APPROVED" | "REJECTED" | "REVOKED";
export type DeletionAction = "ACCEPT" | "REJECT" | "REVOKE" | "REOPEN";
export type AdminResource = "images" | "users" | "deletionRequests";

export type AdminQueryInput = {
  offset: number;
  limit: number;
  search: string;
  sortBy: string;
  direction: SortDirection;
  status?: ImageStatus | DeletionStatus;
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
    rating: "none" | "moderate" | "violent";
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

export type AdminDeletion = {
  id: string;
  imageId: string;
  reason: string;
  status: DeletionStatus;
  requester: { id: string; username: string };
  moderator: { id: string; username: string } | null;
  createdAt: string;
  processedAt: string | null;
  imageDeletedAt: string | null;
  activeDeletionId: string | null;
};

export const DELETION_QUERY = `query AdminDeletions($input: DeletionQueryInput) {
  deletionRequests(input: $input) {
    total
    items {
      id imageId reason status createdAt processedAt imageDeletedAt activeDeletionId
      requester { id username }
      moderator { id username }
    }
  }
}`;

export const MODERATE_DELETION = `mutation ModerateDeletion($id: ID!, $action: DeletionAction!, $expectedStatus: DeletionStatus!) {
  moderateDeletionRequest(id: $id, action: $action, expectedStatus: $expectedStatus) { id status }
}`;
