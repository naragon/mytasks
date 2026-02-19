ALTER TABLE tasks ADD COLUMN notes TEXT DEFAULT '' CHECK(length(notes) <= 255);
