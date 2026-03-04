import type { Page } from '@playwright/test';

/**
 * Log in as a user via the login form.
 * Requires the dev server + backend to be running.
 */
export async function loginAs(page: Page, email: string, password: string) {
	await page.goto('/login');
	await page.getByLabel(/email/i).fill(email);
	await page.getByLabel(/password/i).fill(password);
	await page.getByRole('button', { name: /sign in|log in/i }).click();
	// Wait for redirect away from /login
	await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 5000 });
}
