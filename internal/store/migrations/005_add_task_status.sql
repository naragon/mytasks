ALTER TABLE tasks ADD COLUMN status TEXT NOT NULL DEFAULT 'todo' CHECK(status IN ('todo', 'in_progress', 'done'));

UPDATE tasks SET status = 'done' WHERE completed = TRUE;
UPDATE tasks SET status = 'todo' WHERE completed = FALSE;

CREATE INDEX IF NOT EXISTS idx_tasks_project_status ON tasks(project_id, status, sort_order);
