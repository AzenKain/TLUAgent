ALTER TABLE llm_providers ADD COLUMN timeout_seconds INTEGER NOT NULL DEFAULT 60;
ALTER TABLE llm_providers ADD COLUMN max_retries INTEGER NOT NULL DEFAULT 3;
ALTER TABLE llm_providers ADD COLUMN retry_initial_wait_ms INTEGER NOT NULL DEFAULT 500;
ALTER TABLE llm_providers ADD COLUMN retry_max_wait_ms INTEGER NOT NULL DEFAULT 5000;
ALTER TABLE llm_providers ADD COLUMN allow_private_networks INTEGER NOT NULL DEFAULT 0;
