INSERT INTO llm_chains (id, chain_type, name, description, is_active, failure_threshold, cooldown_seconds, retry_count)
SELECT 'default-audit-chain', 'audit', 'Supersession Audit Chain',
       'Dedicated model chain for the regulation supersession audit agent. Add a chat model here; while it has no nodes the audit uses the default chat chain.',
       1, 3, 300, 3
WHERE NOT EXISTS (SELECT 1 FROM llm_chains WHERE chain_type = 'audit');
