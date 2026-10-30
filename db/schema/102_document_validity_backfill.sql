UPDATE documents SET effective_from = publish_date
 WHERE (effective_from IS NULL OR effective_from = '')
   AND publish_date IS NOT NULL AND publish_date <> '';

UPDATE documents SET validity_tier = 'OFFICIAL'
 WHERE (validity_tier IS NULL OR validity_tier = '')
   AND type IN ('QUY_CHE','QUY_DINH','QUYET_DINH','HUONG_DAN','SO_TAY','CTDT','KE_HOACH','DE_AN');
