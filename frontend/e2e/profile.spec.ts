import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers";

/**
 * Profile tests require a running backend with a seeded test user.
 * Set environment variables TEST_EMAIL and TEST_PASSWORD to run these.
 * Skip if not provided.
 */

const TEST_EMAIL = process.env.TEST_EMAIL;
const TEST_PASSWORD = process.env.TEST_PASSWORD;

test.describe("Profile", () => {
  test.beforeEach(async ({ page }) => {
    if (!TEST_EMAIL || !TEST_PASSWORD) {
      test.skip();
    }
    await loginAs(page, TEST_EMAIL!, TEST_PASSWORD!);
  });

  test("profile page loads with user data", async ({ page }) => {
    await page.goto("/profile");
    // Display name or email should be visible on the profile page
    await expect(
      page
        .locator('input[name="display_name"], [data-testid="display-name"]')
        .first(),
    ).toBeVisible();
  });

  test("profile page shows user email", async ({ page }) => {
    await page.goto("/profile");
    await expect(page.getByText(TEST_EMAIL!)).toBeVisible();
  });
});
