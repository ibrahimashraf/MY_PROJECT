-- Migration 0048 down: Anomaly Detection for Short Links
-- Drops tables for anomaly rules and alerts

DROP TRIGGER IF EXISTS trigger_update_anomaly_alert_updated_at ON anomaly_alerts;
DROP FUNCTION IF EXISTS update_anomaly_alert_updated_at();
DROP TABLE IF EXISTS anomaly_alerts;

DROP TRIGGER IF EXISTS trigger_update_anomaly_rule_updated_at ON anomaly_rules;
DROP FUNCTION IF EXISTS update_anomaly_rule_updated_at();
DROP TABLE IF EXISTS anomaly_rules;