UPDATE documents SET validity_tier = 'SEMESTER_DYNAMIC'
 WHERE (validity_tier IS NULL OR validity_tier = '') AND type = 'THONG_BAO';

UPDATE documents SET validity_tier = 'CURRENT_OPERATIONAL'
 WHERE (validity_tier IS NULL OR validity_tier = '') AND type IN ('TIN_TUC', 'STATIC', 'HUONG_DAN');

UPDATE documents SET validity_tier = 'OFFICIAL'
 WHERE (validity_tier IS NULL OR validity_tier = '')
   AND type IN ('QUY_CHE', 'QUY_DINH', 'QUYET_DINH', 'CTDT', 'KE_HOACH', 'DE_AN', 'SO_TAY');
