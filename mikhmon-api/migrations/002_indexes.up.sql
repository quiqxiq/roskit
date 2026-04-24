CREATE INDEX IF NOT EXISTS idx_voucher_sales_router_sold_at ON voucher_sales(router_id, sold_at);
CREATE INDEX IF NOT EXISTS idx_voucher_sales_profile_name ON voucher_sales(profile_name);

CREATE INDEX IF NOT EXISTS idx_audit_logs_action_created_at ON audit_logs(action, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_routers_session_name ON routers(session_name) WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_profile_price_mappings_profile_name ON profile_price_mappings(profile_name, router_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_system_users_username ON system_users(username) WHERE deleted_at IS NULL;
