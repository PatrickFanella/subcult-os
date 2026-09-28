import type { ConfigContext, ExpoConfig } from 'expo/config';

// Keep in sync with appLinkPaths in backend/internal/app/app_links.go.
export const identityLinkPaths = ['/verify-email', '/recover-password'] as const;

const hostnamePattern = /^(?=.{1,253}$)([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$/;

// withIdentityAppLinks lets verification and recovery emails open the app.
// The host is the PUBLIC_WEB_URL hostname, supplied at build time; builds
// without it keep only the custom scheme and those links stay in the browser.
export function withIdentityAppLinks(config: ExpoConfig, rawHost: string | undefined): ExpoConfig {
	const host = rawHost?.trim().toLowerCase();
	if (!host) return config;
	if (!hostnamePattern.test(host)) {
		throw new Error(`SUBCULT_APP_LINK_HOST must be a hostname such as subcults.subcult.tv, got "${rawHost}"`);
	}
	return {
		...config,
		ios: {
			...config.ios,
			associatedDomains: [...(config.ios?.associatedDomains ?? []), `applinks:${host}`],
		},
		android: {
			...config.android,
			intentFilters: [
				...(config.android?.intentFilters ?? []),
				{
					action: 'VIEW',
					autoVerify: true,
					data: identityLinkPaths.map((pathPrefix) => ({ scheme: 'https', host, pathPrefix })),
					category: ['BROWSABLE', 'DEFAULT'],
				},
			],
		},
	};
}

export default ({ config }: ConfigContext): ExpoConfig =>
	withIdentityAppLinks(config as ExpoConfig, process.env.SUBCULT_APP_LINK_HOST);
