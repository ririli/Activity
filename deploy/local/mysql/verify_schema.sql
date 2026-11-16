-- Run on each instance. Expect only its own business schema: identity 4, core 36, ranking 10.
SELECT TABLE_SCHEMA, COUNT(*) AS table_count,
       SUM(ENGINE = 'InnoDB') AS innodb_count
FROM information_schema.TABLES
WHERE TABLE_SCHEMA IN ('activity_identity', 'activity_core', 'activity_ranking')
  AND TABLE_TYPE = 'BASE TABLE'
GROUP BY TABLE_SCHEMA ORDER BY TABLE_SCHEMA;

-- On mysql-activity, expect four rows per logical family, 16 record tables total.
SELECT TABLE_NAME, ENGINE
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = 'activity_core'
  AND TABLE_NAME REGEXP '^(draw_order|draw_detail|flower_lit_log|wallet_ledger)_[0-3]$'
ORDER BY TABLE_NAME;

-- Every core record table must contain a CHECK using MOD(user_id,4).
SELECT tc.TABLE_NAME, cc.CHECK_CLAUSE, tc.ENFORCED
FROM information_schema.TABLE_CONSTRAINTS AS tc
JOIN information_schema.CHECK_CONSTRAINTS AS cc
  ON cc.CONSTRAINT_SCHEMA = tc.CONSTRAINT_SCHEMA
 AND cc.CONSTRAINT_NAME = tc.CONSTRAINT_NAME
WHERE tc.TABLE_SCHEMA = 'activity_core' AND tc.CONSTRAINT_TYPE = 'CHECK'
  AND cc.CHECK_CLAUSE LIKE '%mod(%'
ORDER BY tc.TABLE_NAME;

-- Must return zero rows.
SELECT TABLE_SCHEMA, TABLE_NAME, CONSTRAINT_NAME
FROM information_schema.TABLE_CONSTRAINTS
WHERE TABLE_SCHEMA IN ('activity_identity', 'activity_core', 'activity_ranking')
  AND CONSTRAINT_TYPE = 'CHECK' AND ENFORCED <> 'YES';
