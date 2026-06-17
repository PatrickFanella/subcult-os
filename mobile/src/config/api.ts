import { Platform } from 'react-native';

const defaultBaseUrl = Platform.select({
  android: 'http://10.0.2.2:38080',
  ios: 'http://localhost:38080',
  default: 'http://localhost:38080',
});

export const apiConfig = {
  baseUrl: process.env.EXPO_PUBLIC_API_URL ?? defaultBaseUrl,
};

export function apiUrl(path: string) {
  return `${apiConfig.baseUrl}${path.startsWith('/') ? path : `/${path}`}`;
}
