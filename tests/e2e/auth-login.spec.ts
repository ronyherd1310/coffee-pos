import { expect, test } from "@playwright/test";

test("cashier can sign in, see the protected shell, and log out", async ({ page }) => {
  let authenticated = false;

  await page.route("**/api/auth/session", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      json: { authenticated },
      status: 200
    });
  });

  await page.route("**/api/auth/login", async (route) => {
    const request = route.request();
    const body = (await request.postDataJSON()) as { pin?: string };

    if (body.pin === "123456") {
      authenticated = true;
      await route.fulfill({ status: 204 });
      return;
    }

    await route.fulfill({
      contentType: "application/json",
      json: { error: "invalid_pin" },
      status: 401
    });
  });

  await page.route("**/api/auth/logout", async (route) => {
    authenticated = false;
    await route.fulfill({ status: 204 });
  });

  await page.goto("/");

  await expect(page.getByRole("heading", { level: 1, name: "Coffee POS" })).toBeVisible();
  await expect(page.getByText("Cashier PIN")).toBeVisible();

  await page.getByLabel("Cashier PIN").fill("000000");
  await page.getByRole("button", { name: "Sign In" }).click();
  await expect(page.getByRole("alert")).toContainText("Invalid PIN. Try again.");
  await expect(page.getByText("Protected POS shell")).toBeHidden();

  await page.getByLabel("Cashier PIN").fill("123456");
  await page.getByRole("button", { name: "Sign In" }).click();

  await expect(page.getByText("Protected POS shell")).toBeVisible();
  await expect(page.getByRole("link", { name: "New Order" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Daily Summary" })).toBeVisible();

  await page.getByRole("button", { name: "Logout" }).click();

  await expect(page.getByText("Cashier PIN")).toBeVisible();
  await expect(page.getByText("Protected POS shell")).toBeHidden();
});
