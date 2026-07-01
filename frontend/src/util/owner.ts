/**
 * Display label for a file's owner. Ownership is single-user for now (every file
 * you can see is your own), so the owner is always the current user.
 */
export function ownerLabel(): string {
  return 'me';
}
