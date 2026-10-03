import { describe, expect, it } from 'vitest';

import {
	emptyPublicRoleApplicationDraft,
	publicRoleApplicationStatusCopy,
	publicRoleApplicationButtonLabel,
	publicRoleAvailabilityLabel,
	publicRoleCanSubmit,
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

	it('describes public role availability and submit state', () => {
		expect(publicRoleAvailabilityLabel(0)).toBe('Open application');
		expect(publicRoleAvailabilityLabel(2)).toBe('2 spots total');
		expect(publicRoleAvailabilityLabel(2, 1)).toBe('1 spot open');
		expect(publicRoleAvailabilityLabel(2, 2)).toBe('Role full');
		expect(publicRoleApplicationStatusCopy(true, null)).toBe('Application sent. The host reviews it from their workspace.');
		expect(publicRoleApplicationStatusCopy(false, 'Please enter your name.')).toBe('Please enter your name.');
		expect(publicRoleCanSubmit(false, false)).toBe(true);
		expect(publicRoleCanSubmit(true, false)).toBe(false);
		expect(publicRoleCanSubmit(false, true)).toBe(false);
	});

	it('validates application drafts', () => {
		expect(emptyPublicRoleApplicationDraft()).toMatchObject({ submitting: false, submitted: false, error: null });
		expect(validatePublicRoleApplicationDraft({ applicantName: '  ', applicantEmail: 'guest@example.com', message: '' })).toBe('Please enter your name.');
		expect(validatePublicRoleApplicationDraft({ applicantName: 'Guest', applicantEmail: 'guest', message: '' })).toBe('Please enter a valid email address.');
		expect(validatePublicRoleApplicationDraft({ applicantName: 'Alex', applicantEmail: 'alex@example.test', message: 'a'.repeat(2001) })).toBe('Message must be 2000 characters or fewer.');
		expect(validatePublicRoleApplicationDraft({ applicantName: 'Alex', applicantEmail: 'alex@example.test', message: 'Happy to help.' })).toBeNull();
	});
});
