UPDATE kg_edges SET status = 'ACTIVE' WHERE status IS NULL OR status = '';
