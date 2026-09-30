import { describe, expect, it } from 'vitest';
import { emptyRoleDraft, rolePayload } from './roleDraft';

describe('participation role draft', () => {
  it('sends an explicit private default instead of the API public default', () => {
    expect(rolePayload({ ...emptyRoleDraft(), name: '  Door helper  ', description: '  Welcome people  ' })).toEqual({
      name: 'Door helper', description: 'Welcome people', capacity: 0, public: false,
    });
    expect(rolePayload({ ...emptyRoleDraft(), name: 'Performer', capacity: '2', public: true })).toMatchObject({ capacity: 2, public: true });
  });
  it.each(['-1', '1.5', 'Infinity', 'NaN', '2147483648'])('rejects unrepresentable role capacity %s', (capacity) => {
    expect(() => rolePayload({ ...emptyRoleDraft(), name: 'Performer', capacity })).toThrow('whole number');
  });
  it('checks the editable description boundary in Unicode characters', () => {
    expect(rolePayload({ ...emptyRoleDraft(), name: 'Performer', description: '🎶'.repeat(2000) }).description).toHaveLength(4000);
    expect(() => rolePayload({ ...emptyRoleDraft(), name: 'Performer', description: '🎶'.repeat(2001) })).toThrow('2000 characters');
    expect(() => rolePayload(emptyRoleDraft())).toThrow('role name');
  });
});
