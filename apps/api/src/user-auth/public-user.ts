import { User } from '@app/database';

export interface PublicUser {
  id: string;
  telegramId: string | null;
  email: string | null;
  username: string | null;
  firstName: string | null;
  lastName: string | null;
  phone: string | null;
  avatarUrl: string | null;
}

export function toPublicUser(user: User): PublicUser {
  return {
    id: user.id,
    telegramId: user.telegramId,
    email: user.email,
    username: user.username,
    firstName: user.firstName,
    lastName: user.lastName,
    phone: user.phone,
    avatarUrl: user.avatarUrl,
  };
}
