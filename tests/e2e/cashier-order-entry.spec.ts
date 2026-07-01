import { expect, test } from "@playwright/test";

const menuResponse = {
  categories: [
    {
      name: "Coffee",
      slug: "coffee",
      items: [
        {
          name: "Americano",
          slug: "americano",
          priceRp: 18000,
          modifierGroups: [
            {
              name: "Temperature",
              slug: "temperature",
              required: true,
              selectionType: "single",
              options: [
                { name: "Hot", slug: "hot", priceDeltaRp: 0 },
                { name: "Iced", slug: "iced", priceDeltaRp: 2000 }
              ]
            },
            {
              name: "Sugar",
              slug: "sugar",
              required: true,
              selectionType: "single",
              options: [
                { name: "Normal", slug: "normal", priceDeltaRp: 0 },
                { name: "Less", slug: "less", priceDeltaRp: 0 }
              ]
            }
          ]
        },
        {
          name: "Latte",
          slug: "latte",
          priceRp: 25000,
          modifierGroups: []
        }
      ]
    }
  ]
};

test("cashier creates a QRIS paid order with two differently modified lines", async ({ page }) => {
  await page.route("**/api/auth/session", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      json: { authenticated: true },
      status: 200
    });
  });

  await page.route("**/api/auth/login", async (route) => {
    await route.fulfill({ status: 204 });
  });

  await page.route("**/api/pos/menu", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      json: menuResponse,
      status: 200
    });
  });

  await page.route("**/api/pos/orders", async (route) => {
    const body = (await route.request().postDataJSON()) as {
      lines: unknown[];
      paymentMethod: string;
    };

    expect(body.paymentMethod).toBe("qris");
    expect(body.lines).toHaveLength(2);

    await route.fulfill({
      contentType: "application/json",
      json: {
        orderId: "order-7",
        queueNumber: 7,
        businessDate: "2026-07-01",
        status: "paid",
        paymentMethod: "qris",
        paidAt: "2026-07-01T03:00:00Z",
        cancelledAt: null,
        note: null,
        totalRp: 38000,
        lines: [
          {
            menuItemSlug: "americano",
            menuItemName: "Americano",
            unitPriceRp: 18000,
            quantity: 1,
            lineTotalRp: 18000,
            modifiers: [
              {
                groupSlug: "temperature",
                groupName: "Temperature",
                optionSlug: "hot",
                optionName: "Hot",
                priceDeltaRp: 0
              },
              {
                groupSlug: "sugar",
                groupName: "Sugar",
                optionSlug: "normal",
                optionName: "Normal",
                priceDeltaRp: 0
              }
            ]
          },
          {
            menuItemSlug: "americano",
            menuItemName: "Americano",
            unitPriceRp: 20000,
            quantity: 1,
            lineTotalRp: 20000,
            modifiers: [
              {
                groupSlug: "temperature",
                groupName: "Temperature",
                optionSlug: "iced",
                optionName: "Iced",
                priceDeltaRp: 2000
              },
              {
                groupSlug: "sugar",
                groupName: "Sugar",
                optionSlug: "less",
                optionName: "Less",
                priceDeltaRp: 0
              }
            ]
          }
        ]
      },
      status: 201
    });
  });

  await page.route("**/api/pos/orders/*/cancel", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      json: { error: "order_not_cancellable" },
      status: 409
    });
  });

  await page.goto("/");

  await expect(page.getByRole("heading", { level: 2, name: "New Order" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Print Ticket" })).toBeDisabled();

  await addAmericanoLine(page, "Hot", "Normal");
  await addAmericanoLine(page, "Iced +Rp2.000", "Less");

  await expect(page.getByText("Hot, Normal")).toBeVisible();
  await expect(page.getByText("Iced, Less")).toBeVisible();

  await page.getByLabel("QRIS").click();
  await expect(page.getByAltText("Static QRIS payment code")).toHaveAttribute(
    "src",
    "/qris/static-qris.png"
  );

  await page.getByRole("button", { name: "Confirm Paid" }).click();
  await expect(page.getByRole("dialog", { name: "Confirm payment" })).toBeVisible();
  await page
    .getByRole("dialog", { name: "Confirm payment" })
    .getByRole("button", { name: "Confirm Paid" })
    .click();

  await expect(page.getByText("Queue No. 007")).toBeVisible();
  await expect(page.getByRole("button", { name: "Print Ticket" })).toBeEnabled();
});

async function addAmericanoLine(page: import("@playwright/test").Page, temperature: string, sugar: string) {
  await page.getByRole("button", { name: "Americano Rp18.000" }).click();
  await page.getByLabel(temperature).click();
  await page.getByLabel(sugar).click();
  await page.getByRole("button", { name: "Add Item To Order" }).click();
}
