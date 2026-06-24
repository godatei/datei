import { File } from '~/api/models';

/** Display label for a file's owner: "me" for the current user, else their name. */
export function ownerLabel(file: File, currentUserId: string | undefined): string {
  return file.ownerId === currentUserId ? 'me' : file.ownerName;
}
