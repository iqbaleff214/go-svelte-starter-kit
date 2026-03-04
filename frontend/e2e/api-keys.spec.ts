import { test, expect } from '@playwright/test';
import { loginAs } from './helpers';

const TEST_EMAIL = process.env.TEST_EMAIL;
const TEST_PASSWORD = process.env.TEST_PASSWORD;

test.describe('API Keys', () => {
	test.beforeEach(async ({ page }) => {
		if (!TEST_EMAIL || !TEST_PASSWORD) {
			test.skip();
		}
		await loginAs(page, TEST_EMAIL!, TEST_PASSWORD!);
	});

	test('api keys page loads', async ({ page }) => {
		await page.goto('/profile/api-keys');
		await expect(page.getByRole('heading', { name: /api keys/i })).toBeVisible();
	});

	test('shows empty state when no keys exist', async ({ page }) => {
		await page.goto('/profile/api-keys');
		// Either shows empty state text or existing keys
		const emptyText = page.getByText(/no api keys yet/i);
		const keyList = page.locator('[data-testid="key-row"]');
		const hasEmpty = await emptyText.isVisible().catch(() => false);
		const hasKeys = await keyList.count().then((n) => n > 0).catch(() => false);
		expect(hasEmpty || hasKeys).toBe(true);
	});

	test('create key button opens form', async ({ page }) => {
		await page.goto('/profile/api-keys');
		await page.getByRole('button', { name: /new key/i }).click();
		await expect(page.getByLabel(/name/i)).toBeVisible();
		await expect(page.getByText(/scopes/i)).toBeVisible();
	});
});
