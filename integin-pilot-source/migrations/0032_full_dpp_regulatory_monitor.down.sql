-- 0032_full_dpp_regulatory_monitor.down.candidate.sql
-- Reverse: Drop DPP 4 Pillars + Regulatory Monitor + CTRN Concept

DROP TRIGGER IF EXISTS trg_compliance_action_updated_at ON compliance_action;
DROP TRIGGER IF EXISTS trg_regulatory_monitor_updated_at ON regulatory_monitor;
DROP TRIGGER IF EXISTS trg_product_passport_dpp_updated_at ON product_passport_dpp;

DROP TRIGGER IF EXISTS trg_prevent_immutable_dpp_delete ON product_passport_dpp;
DROP FUNCTION IF EXISTS prevent_immutable_dpp_delete();

DROP TRIGGER IF EXISTS trg_prevent_immutable_dpp_update ON product_passport_dpp;
DROP FUNCTION IF EXISTS prevent_immutable_dpp_update();

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS compliance_action;
DROP TABLE IF EXISTS regulatory_monitor;
DROP TABLE IF EXISTS product_passport_dpp;