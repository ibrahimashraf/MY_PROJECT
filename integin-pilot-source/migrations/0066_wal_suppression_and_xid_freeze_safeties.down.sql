BEGIN;

ALTER TABLE river_job RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_vacuum_threshold,
    autovacuum_vacuum_cost_limit,
    autovacuum_vacuum_cost_delay,
    autovacuum_freeze_max_age,
    autovacuum_freeze_table_age,
    autovacuum_freeze_min_age
);

ALTER TABLE river_notification RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_vacuum_threshold,
    autovacuum_vacuum_cost_limit,
    autovacuum_vacuum_cost_delay
);

COMMIT;
