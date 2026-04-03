// @ts-check
const { test, expect } = require('@playwright/test');

async function ensureProjectPage(page, projectName) {
  await page.goto('/');
  if (page.url().includes('/projects/')) {
    return;
  }

  await page.locator('.sidebar button:has-text("+ New")').click();
  await page.locator('#new-project-form input[name="name"]').fill(projectName);
  await page.locator('#new-project-form button[type="submit"]').click();
  await page.waitForURL(/\/projects\/\d+/);
}

test.describe('Home page', () => {
  test('redirects to first project or shows empty state', async ({ page }) => {
    const response = await page.goto('/');
    // Either redirected to a project or shows empty state
    const url = page.url();
    const isProject = url.includes('/projects/');
    const isRoot = url === 'http://localhost:3000/' || url === 'http://localhost:3000';
    expect(isProject || isRoot).toBeTruthy();
  });

  test('empty state has new project button', async ({ page }) => {
    await page.goto('/');
    const url = page.url();
    if (!url.includes('/projects/')) {
      await expect(page.locator('text=Create your first project')).toBeVisible();
      await expect(page.locator('button:has-text("New Project")')).toBeVisible();
    }
  });
});

test.describe('Sidebar', () => {
  test('sidebar is present on home page', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('.sidebar')).toBeVisible();
  });

  test('sidebar has upcoming and archive links', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('.sidebar a[href="/upcoming"]')).toBeVisible();
    await expect(page.locator('.sidebar a[href="/archive"]')).toBeVisible();
  });

  test('sidebar can collapse and expand', async ({ page }) => {
    await page.goto('/');

    const layout = page.locator('.app-layout').first();
    const sidebar = page.locator('.sidebar').first();
    const toggle = page.locator('[data-action="toggle-sidebar"]').first();

    const expandedBox = await sidebar.boundingBox();
    expect(expandedBox).not.toBeNull();

    await toggle.click();
    await expect(layout).toHaveClass(/sidebar-collapsed/);

    const collapsedBox = await sidebar.boundingBox();
    expect(collapsedBox).not.toBeNull();
    expect(collapsedBox.width).toBeLessThan(expandedBox.width);

    await toggle.click();
    await expect(layout).not.toHaveClass(/sidebar-collapsed/);
  });

  test('sidebar can be resized', async ({ page }) => {
    await page.setViewportSize({ width: 1400, height: 900 });
    await page.goto('/');

    const sidebar = page.locator('.sidebar').first();
    const initialBox = await sidebar.boundingBox();
    expect(initialBox).not.toBeNull();

    const widenButton = page.locator('[data-action="widen-sidebar"]').first();
    await widenButton.click();
    await widenButton.click();
    await widenButton.click();

    const resizedBox = await sidebar.boundingBox();
    expect(resizedBox).not.toBeNull();
    expect(resizedBox.width).toBeGreaterThan(initialBox.width + 40);

    const storedWidth = await page.evaluate(() => localStorage.getItem('mytasks.sidebar.width'));
    expect(storedWidth).not.toBeNull();
  });
});

test.describe('Project creation and Kanban board', () => {
  test('new project button opens form without javascript errors', async ({ page }) => {
    const pageErrors = [];
    page.on('pageerror', error => pageErrors.push(error.message));

    await page.goto('/');
    await page.locator('.sidebar button:has-text("+ New")').click();
    await expect(page.locator('#new-project-form')).toBeVisible();
    expect(pageErrors).not.toContain(expect.stringContaining('showProjectForm is not defined'));
  });

  test('can create a project and see kanban board', async ({ page }) => {
    await page.goto('/');

    // Open new project form
    await page.locator('.sidebar button:has-text("+ New")').click();

    // Fill in project name
    await page.locator('#new-project-form input[name="name"]').fill('Test Project E2E');
    await page.locator('#new-project-form button[type="submit"]').click();

    // Should land on kanban board for the new project
    await page.waitForURL(/\/projects\/\d+/);
    await expect(page.locator('.kanban-board')).toBeVisible();
    await expect(page.locator('.kanban-column')).toHaveCount(3);
  });

  test('kanban board has three columns', async ({ page }) => {
    await page.goto('/');
    if (page.url().includes('/projects/')) {
      await expect(page.locator('.kanban-column[data-status="todo"]')).toBeVisible();
      await expect(page.locator('.kanban-column[data-status="in_progress"]')).toBeVisible();
      await expect(page.locator('.kanban-column[data-status="done"]')).toBeVisible();
    }
  });

  test('kanban columns render side by side on desktop', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await ensureProjectPage(page, 'Kanban Layout Project');

    const todoBox = await page.locator('.kanban-column[data-status="todo"]').boundingBox();
    const inProgressBox = await page.locator('.kanban-column[data-status="in_progress"]').boundingBox();
    const doneBox = await page.locator('.kanban-column[data-status="done"]').boundingBox();

    expect(todoBox).not.toBeNull();
    expect(inProgressBox).not.toBeNull();
    expect(doneBox).not.toBeNull();

    expect(Math.abs(todoBox.y - inProgressBox.y)).toBeLessThan(20);
    expect(Math.abs(inProgressBox.y - doneBox.y)).toBeLessThan(20);
    expect(inProgressBox.x).toBeGreaterThan(todoBox.x);
    expect(doneBox.x).toBeGreaterThan(inProgressBox.x);
  });
});

test.describe('Task creation', () => {
  test('can add a task to the todo column', async ({ page }) => {
    await page.goto('/');

    // Navigate to a project if not already there
    if (!page.url().includes('/projects/')) {
      // Create a project first
      await page.locator('.sidebar button:has-text("+ New")').click();
      await page.locator('#new-project-form input[name="name"]').fill('Task Test Project');
      await page.locator('#new-project-form button[type="submit"]').click();
      await page.waitForURL(/\/projects\/\d+/);
    }

    // Click + button in todo column
    await page.locator('.kanban-column[data-status="todo"] button:has-text("+")').click();

    // Fill in task description
    const taskForm = page.locator('#kanban-form-todo');
    await expect(taskForm).toBeVisible();
    await taskForm.locator('input[name="description"], textarea[name="description"]').fill('My E2E Test Task');
    await taskForm.locator('button[type="submit"]').click();

    // Task should appear in todo column
    await expect(page.locator('.kanban-column[data-status="todo"] .kanban-card:has-text("My E2E Test Task")').first()).toBeVisible();

    // All three columns should remain visible after task creation
    await expect(page.locator('.kanban-column[data-status="todo"]')).toBeVisible();
    await expect(page.locator('.kanban-column[data-status="in_progress"]')).toBeVisible();
    await expect(page.locator('.kanban-column[data-status="done"]')).toBeVisible();
  });
});

test.describe('Upcoming view', () => {
  test('upcoming page loads', async ({ page }) => {
    await page.goto('/upcoming');
    await expect(page).toHaveTitle(/Upcoming/);
    await expect(page.locator('.sidebar')).toBeVisible();
    await expect(page.locator('h2:has-text("Upcoming")')).toBeVisible();
  });

  test('upcoming has day filter buttons', async ({ page }) => {
    await page.goto('/upcoming');
    await expect(page.locator('a[href="/upcoming?days=7"]')).toBeVisible();
    await expect(page.locator('a[href="/upcoming?days=14"]')).toBeVisible();
    await expect(page.locator('a[href="/upcoming?days=30"]')).toBeVisible();
  });

  test('upcoming filter changes active button', async ({ page }) => {
    await page.goto('/upcoming?days=14');
    await expect(page.locator('a[href="/upcoming?days=14"].btn-primary')).toBeVisible();
    await expect(page.locator('a[href="/upcoming?days=7"].btn-secondary')).toBeVisible();
  });
});

test.describe('Archive view', () => {
  test('archive page loads', async ({ page }) => {
    await page.goto('/archive');
    await expect(page).toHaveTitle(/Archive/);
    await expect(page.locator('.sidebar')).toBeVisible();
    await expect(page.locator('h2:has-text("Archived Projects")')).toBeVisible();
  });

  test('archive shows empty state when no completed projects', async ({ page }) => {
    await page.goto('/archive');
    // Either shows project cards or empty state
    const hasProjects = await page.locator('.archive-card').count();
    if (hasProjects === 0) {
      await expect(page.locator('.empty-state')).toBeVisible();
    }
  });
});

test.describe('Project complete and archive flow', () => {
  test('completing a project moves it to archive', async ({ page }) => {
    // Create a project
    await page.goto('/');
    await page.locator('.sidebar button:has-text("+ New")').click();
    await page.locator('#new-project-form input[name="name"]').fill('Complete Me Project');
    await page.locator('#new-project-form button[type="submit"]').click();
    await page.waitForURL(/\/projects\/\d+/);

    // Accept the confirm dialog and click complete
    page.once('dialog', dialog => dialog.accept());
    await page.locator('.kanban-header button:has-text("Complete")').click();
    await page.waitForURL('/archive');

    // Project should be in archive
    await expect(page.locator('.archive-card:has-text("Complete Me Project")').first()).toBeVisible();

    // Project should not be in sidebar active list
    await expect(page.locator('.sidebar .sidebar-item:has-text("Complete Me Project")')).toHaveCount(0);
  });
});
