import type { User } from '@app/database';

// The subset of a user that's safe to show next to their public content —
// never email/phone.
export interface PublicAuthor {
  id: string;
  firstName: string | null;
  username: string | null;
  avatarUrl: string | null;
}

export function toPublicAuthor(user: User): PublicAuthor {
  return {
    id: user.id,
    firstName: user.firstName,
    username: user.username,
    avatarUrl: user.avatarUrl,
  };
}
