export type Wrapped<T> = {
  error?: string | undefined;
  data: T
}

export type Rating = 'none' | 'moderate' | 'violent';

export type Tag = {
  id: number,
  name: string
}

export type Image = {
  id: number;
  imageKey: string;
  imageUrl: string;
  currentVersionId: number;
  userId: number;
  createdAt: string;
};

export type Version = {
  id: number;
  createdAt: string;
  version: number;
  text: string;
  rating: Rating;
  userId: number;
}

export type Preferences = {
  hideNSFW?: boolean,
  hideViolent?: boolean,
}

export type UserIdentifier = {
  id: number,
  username: string,
  createdAt: string,
}

export type Self = UserIdentifier & {
  email: string,
  passwordHash: string,
  preference: Preferences,
  permission: number,
}

export type Identity = Self & {
  authorizedVia: string,
}


export type SignResponse = {
  sessionId: number,
  key: string,
  url: string
}

export type AckResponse = {
  id: number,
}

export type Favorite = {
  imageID: number,
  userID: number,
  shortcut: string | null,
  createdAt: string,
}

export type AppKey = {
  id: number,
  label: string,
  createdAt: string,
  permission: number,
  lastActivatedAt: string | null,
}