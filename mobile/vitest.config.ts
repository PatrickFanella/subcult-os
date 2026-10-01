import { defineConfig } from 'vitest/config';

export default defineConfig({
	test: {
		environment: 'node',
		include: ['src/modules/**/*.test.ts', 'src/theme/**/*.test.ts'],
	},
});
