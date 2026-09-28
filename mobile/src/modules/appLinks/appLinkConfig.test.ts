import { describe, expect, it } from 'vitest';

import type { ExpoConfig } from 'expo/config';

import { withIdentityAppLinks } from '../../../app.config';

const base: ExpoConfig = {
	name: 'SUBCULT OS',
	slug: 'subcult-os-mobile',
	scheme: 'subcultos',
	ios: { bundleIdentifier: 'tv.clpr.subcultos' },
	android: { package: 'tv.clpr.subcultos' },
};

describe('withIdentityAppLinks', () => {
	it('leaves the config unchanged without a host', () => {
		expect(withIdentityAppLinks(base, undefined)).toBe(base);
		expect(withIdentityAppLinks(base, '  ')).toBe(base);
	});

	it('claims only identity email paths on the configured host', () => {
		const config = withIdentityAppLinks(base, ' Subcults.Subcult.TV ');
		expect(config.scheme).toBe('subcultos');
		expect(config.ios?.bundleIdentifier).toBe('tv.clpr.subcultos');
		expect(config.ios?.associatedDomains).toEqual(['applinks:subcults.subcult.tv']);
		expect(config.android?.package).toBe('tv.clpr.subcultos');
		expect(config.android?.intentFilters).toEqual([
			{
				action: 'VIEW',
				autoVerify: true,
				data: [
					{ scheme: 'https', host: 'subcults.subcult.tv', pathPrefix: '/verify-email' },
					{ scheme: 'https', host: 'subcults.subcult.tv', pathPrefix: '/recover-password' },
				],
				category: ['BROWSABLE', 'DEFAULT'],
			},
		]);
	});

	it('rejects URLs, ports and wildcard hosts', () => {
		for (const host of ['https://subcults.subcult.tv', 'subcults.subcult.tv:443', '*.subcult.tv', 'localhost']) {
			expect(() => withIdentityAppLinks(base, host)).toThrow(/SUBCULT_APP_LINK_HOST/);
		}
	});
});
