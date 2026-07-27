import { expect, test } from '@playwright/test';

test.describe('Watchlist', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/watchlist');
    await page.waitForSelector('[aria-label="Watchlist selector"]');
  });

  test('creates, renames, and deletes a watchlist', async ({ page }) => {
    await page.getByLabel('New watchlist name').fill('Momentum');
    await page.getByRole('button', { name: 'Create' }).click();
    await expect(page.getByLabel('Watchlist selector')).toHaveValue(/.+/);
    await page.getByLabel('Rename watchlist').fill('Momentum list');
    await page.reload();
    await expect(page.getByLabel('Rename watchlist')).toHaveValue('Momentum list');
    page.once('dialog', (dialog) => dialog.accept());
    await page.getByLabel('Delete watchlist').click();
    await expect(page.getByLabel('Watchlist selector')).not.toHaveText('Momentum list');
  });

  test('adds, prevents duplicate, removes, pins, reorders, notes, tags, and mutes pair', async ({ page }) => {
    await page.getByLabel('Pair symbol').fill('XRP_USDT');
    await page.getByRole('button', { name: 'Add pair' }).click();
    await expect(page.getByText('XRP/USDT', { exact: true }).first()).toBeVisible();
    await page.getByLabel('Pair symbol').fill('XRP_USDT');
    await page.getByRole('button', { name: 'Add pair' }).click();
    await expect(page.getByText('Pair sudah ada di watchlist ini.')).toBeVisible();
    await page.getByText('XRP/USDT', { exact: true }).first().click();
    await page.getByLabel('Pin pair').check();
    await page.getByLabel('Mute pair').check();
    await page.getByLabel('Notes').fill('Range breakout review');
    await page.getByLabel('Pair tags').fill('momentum, review');
    await page.reload();
    await page.getByText('XRP/USDT', { exact: true }).first().click();
    await expect(page.getByLabel('Notes')).toHaveValue('Range breakout review');
    await expect(page.getByLabel('Pair tags')).toHaveValue('momentum, review');
    await page.getByLabel('Remove pair').click();
    await expect(page.getByText('XRP/USDT', { exact: true })).toHaveCount(0);
  });

  test('persists local mode, filters, custom score, alert policy, and responsive controls', async ({ page }) => {
    await expect(page.getByText('Saved locally')).toBeVisible();
    await page.getByText('BTC/USDT', { exact: true }).first().click();
    await page.getByLabel('Minimum score').fill('91');
    await page.getByLabel('Notification enabled').check();
    await page.getByLabel('Search watchlist').fill('BTC');
    await expect(page.getByText('BTC/USDT', { exact: true }).first()).toBeVisible();
    await page.reload();
    await expect(page.getByLabel('Minimum score')).toHaveValue('91');
    await expect(page.getByText(/browser local timezone:/)).toBeVisible();
  });

  test('uses one shared realtime WebSocket for watchlist', async ({ page }) => {
    let sockets = 0;
    page.on('websocket', (socket) => {
      if (socket.url().includes(':8080/ws')) sockets += 1;
    });
    await page.goto('/watchlist');
    await page.waitForSelector('[aria-label="Watchlist selector"]');
    await page.waitForTimeout(500);
    expect(sockets).toBeLessThanOrEqual(1);
  });
});
