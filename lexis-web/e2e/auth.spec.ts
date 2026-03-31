import { test, expect } from "@playwright/test";

test.describe("Auth pages", () => {
  test("login page renders", async ({ page }) => {
    await page.goto("/login");
    await expect(page.locator("text=lang.tutor")).toBeVisible();
    await expect(page.locator("text=Вход")).toBeVisible();
    await expect(page.locator('input[type="email"]')).toBeVisible();
    await expect(page.locator('input[type="password"]')).toBeVisible();
    await expect(page.locator("text=[ войти ]")).toBeVisible();
  });

  test("register page renders", async ({ page }) => {
    await page.goto("/register");
    await expect(page.locator("text=Регистрация")).toBeVisible();
    await expect(page.locator("text=[ создать аккаунт ]")).toBeVisible();
  });

  test("login page has link to register", async ({ page }) => {
    await page.goto("/login");
    await expect(page.locator("text=Регистрация")).toBeVisible();
  });

  test("register page has link to login", async ({ page }) => {
    await page.goto("/register");
    await expect(page.locator("text=Войти")).toBeVisible();
  });

  test("unauthenticated user redirected to login", async ({ page }) => {
    await page.goto("/chat");
    await page.waitForURL("**/login");
    await expect(page).toHaveURL(/login/);
  });
});
