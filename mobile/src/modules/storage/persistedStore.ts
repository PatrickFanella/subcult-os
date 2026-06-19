import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';

type PersistedStorage = {
	getItem(key: string): Promise<string | null>;
	setItem(key: string, value: string): Promise<void>;
	removeItem(key: string): Promise<void>;
};

async function readWebValue(key: string) {
	try {
		return globalThis.localStorage?.getItem(key) ?? null;
	} catch {
		return null;
	}
}

async function writeWebValue(key: string, value: string) {
	try {
		globalThis.localStorage?.setItem(key, value);
	} catch {
		return;
	}
}

async function deleteWebValue(key: string) {
	try {
		globalThis.localStorage?.removeItem(key);
	} catch {
		return;
	}
}

const storage: PersistedStorage = Platform.OS === 'web'
	? {
		getItem: readWebValue,
		setItem: writeWebValue,
		removeItem: deleteWebValue,
	}
	: {
		getItem: (key) => SecureStore.getItemAsync(key),
		setItem: (key, value) => SecureStore.setItemAsync(key, value),
		removeItem: (key) => SecureStore.deleteItemAsync(key),
	};

export async function loadPersistedValue(key: string) {
	return storage.getItem(key);
}

export async function savePersistedValue(key: string, value: string) {
	await storage.setItem(key, value);
}

export async function removePersistedValue(key: string) {
	await storage.removeItem(key);
}
