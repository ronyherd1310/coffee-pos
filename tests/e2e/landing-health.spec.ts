import { expect, test } from "@playwright/test";

test("landing page shows backend health", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByRole("heading", { level: 1, name: "Coffee POS" })).toBeVisible();
  await expect(page.getByText("coffee-pos-backend")).toBeVisible();
  await expect(page.getByText("ok")).toBeVisible();
});
