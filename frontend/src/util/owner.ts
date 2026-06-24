import { File } from '~/api/models';

/** Display label for a file's owner: "me" for the current user, else their name. */
export function ownerLabel(file: File, currentUserId: string | undefined): string {
  if (!file.ownerId || file.ownerId === currentUserId) return 'me';
  return file.ownerName ?? file.ownerId;
}
