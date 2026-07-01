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
          imagePath: "/menu/americano.png",
          popularityRank: 1,
          bestSeller: true,
          promo: false,
          iced: true,
          lowSugar: true,
          newArrival: false,
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
          imagePath: "/menu/latte.png",
          popularityRank: 2,
          bestSeller: false,
          promo: false,
          iced: false,
          lowSugar: false,
          newArrival: false,
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
      json: {
        orderId: "order-7",
        queueNumber: 7,
        businessDate: "2026-07-01",
        status: "cancelled",
        paymentMethod: "qris",
        paidAt: "2026-07-01T03:00:00Z",
        cancelledAt: "2026-07-01T04:00:00Z",
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
      status: 200
    });
  });

  await page.goto("/");

  await expect(page.getByRole("heading", { level: 2, name: "New Order" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Print Ticket" })).toHaveCount(0);

  await addAmericanoLine(page, "Hot", "Normal");
  await page.getByPlaceholder("Search menu item...").fill("latte");
  await expect(page.getByText("Hot, Normal")).toBeVisible();
  await page.getByRole("button", { name: "Best Seller" }).click();
  await expect(page.getByText("Hot, Normal")).toBeVisible();
  await page.getByPlaceholder("Search menu item...").fill("");
  await page.getByRole("button", { name: "Best Seller" }).click();

  await addAmericanoLine(page, "Iced +Rp2.000", "Less");

  await expect(page.getByText("Hot, Normal")).toBeVisible();
  await expect(page.getByText("Iced, Less")).toBeVisible();

  await page.getByLabel("QRIS").click();
  await expect(page.getByAltText("Static QRIS payment code")).toBeHidden();

  await page.getByRole("button", { name: "Proceed to Payment" }).click();
  const dialog = page.getByRole("dialog", { name: "Payment: QRIS" });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByAltText("Static QRIS payment code")).toHaveAttribute(
    "src",
    "/qris/static-qris.png"
  );

  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(page.getByRole("dialog", { name: "Payment: QRIS" })).toBeHidden();
  await expect(page.getByText("Hot, Normal")).toBeVisible();
  await expect(page.getByText("Iced, Less")).toBeVisible();

  await page.getByRole("button", { name: "Proceed to Payment" }).click();
  await page
    .getByRole("dialog", { name: "Payment: QRIS" })
    .getByRole("button", { name: "Confirm Paid" })
    .click();

  await expect(page.getByText("Queue No. 007")).toBeVisible();
  await expect(page.getByRole("button", { name: "Print Ticket" })).toBeEnabled();

  await page.getByRole("button", { name: "Cancel Order" }).click();
  await page
    .getByRole("dialog", { name: "Cancel order" })
    .getByRole("button", { name: "Cancel Order" })
    .click();

  await expect(page.getByText("Status: Cancelled")).toBeVisible();
  await expect(page.getByRole("button", { name: "Print Ticket" })).toBeDisabled();
});

async function addAmericanoLine(page: import("@playwright/test").Page, temperature: string, sugar: string) {
  await page.getByRole("button", { name: "Americano Rp18.000" }).click();
  await page.getByLabel(temperature).click();
  await page.getByLabel(sugar).click();
  await page.getByRole("button", { name: "Add Item To Order" }).click();
}
