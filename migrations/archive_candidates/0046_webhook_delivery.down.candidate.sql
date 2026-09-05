-- Drop webhook_deliveries table
DROP TRIGGER IF EXISTS trigger_update_webhook_delivery_updated_at ON webhook_deliveries;
DROP FUNCTION IF EXISTS update_webhook_delivery_updated_at();
DROP TABLE IF EXISTS webhook_deliveries;