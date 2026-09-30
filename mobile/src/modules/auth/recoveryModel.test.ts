import { describe, expect, it } from 'vitest';

import { newPasswordProblem, recoveryRequestNotice } from './recoveryModel';

describe('newPasswordProblem', () => {
	it('requires the API minimum length', () => {
		expect(newPasswordProblem('short', 'short')).toMatch(/at least 8/);
	});

	it('requires a matching confirmation', () => {
		expect(newPasswordProblem('long enough', 'long enougH')).toMatch(/do not match/);
	});

	it('accepts a matching password of the minimum length', () => {
		expect(newPasswordProblem('12345678', '12345678')).toBeNull();
	});
});

describe('recoveryRequestNotice', () => {
	it('does not claim the address has an account', () => {
		const notice = recoveryRequestNotice('person@example.test');
		expect(notice).toMatch(/^If person@example\.test belongs to a verified account/);
		expect(notice).toContain('Email delivery is not confirmed.');
		expect(notice).not.toMatch(/sent|on its way/);
	});
});
