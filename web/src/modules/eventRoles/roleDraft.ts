export type RoleDraft = { name: string; description: string; capacity: string; public: boolean };

export function emptyRoleDraft(): RoleDraft {
  return { name: '', description: '', capacity: '0', public: false };
}

export function rolePayload(draft: RoleDraft) {
  const name = draft.name.trim();
  const description = draft.description.trim();
  const capacity = Number(draft.capacity.trim());
  if (!name) throw new Error('Enter a role name.');
  if (Array.from(description).length > 2000) throw new Error('Role description must be 2000 characters or fewer.');
  // The existing role contract stores capacity in a PostgreSQL integer.
  if (!Number.isInteger(capacity) || capacity < 0 || capacity > 2147483647) {
    throw new Error('Role capacity must be a whole number from 0 to 2147483647.');
  }
  return { name, description, capacity, public: draft.public };
}
