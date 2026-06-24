import { File } from '~/api/models';

/** Display label for a file's owner: "me" for the current user, else the id. */
export function ownerLabel(file: File, currentUserId: string | undefined): string {
  return !file.createdBy || file.createdBy === currentUserId ? 'me' : file.createdBy;
}
