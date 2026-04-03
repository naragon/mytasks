CREATE INDEX IF NOT EXISTS idx_tasks_project_status_completed_at
    ON tasks(project_id, status, completed_at);
