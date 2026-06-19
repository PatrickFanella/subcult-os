import { describe, expect, it } from 'vitest';

import {
	emptyPublicRoleApplicationDraft,
	publicRoleApplicationButtonLabel,
	publicRoleCapacityLabel,
	validatePublicRoleApplicationDraft,
} from './publicEventRolesModel';

describe('publicEventRolesModel', () => {
	it('formats role capacity labels', () => {
		expect(publicRoleCapacityLabel(0)).toBe('Open');
		expect(publicRoleCapacityLabel(1)).toBe('1 spot');
		expect(publicRoleCapacityLabel(3)).toBe('3 spots');
	});

	it('formats application button labels', () => {
		expect(publicRoleApplicationButtonLabel(false, false)).toBe('Submit application');
		expect(publicRoleApplicationButtonLabel(true, false)).toBe('Submitting…');
		expect(publicRoleApplicationButtonLabel(false, true)).toBe('Submitted');
	});

	it('validates application drafts', () => {
		expect(emptyPublicRoleApplicationDraft()).toMatchObject({ submitting: false, submitted: false, error: null });
		expect(validatePublicRoleApplicationDraft({ applicantName: ' ', applicantEmail: 'alex@example.test', message: '' })).toBe('Please enter your name.');
		expect(validatePublicRoleApplicationDraft({ applicantName: 'Alex', applicantEmail: 'invalid', message: '' })).toBe('Please enter a valid email address.');
		expect(validatePublicRoleApplicationDraft({ applicantName: 'Alex', applicantEmail: 'alex@example.test', message: 'a'.repeat(2001) })).toBe('Message must be 2000 characters or fewer.');
		expect(validatePublicRoleApplicationDraft({ applicantName: 'Alex', applicantEmail: 'alex@example.test', message: 'Happy to help.' })).toBeNull();
	});
});
