CREATE TABLE IF NOT EXISTS routers (
    id BIGSERIAL PRIMARY KEY,
    session_name VARCHAR(100) NOT NULL,
    ip VARCHAR(45) NOT NULL,
    username VARCHAR(100) NOT NULL,
    password VARCHAR(500) NOT NULL,
    hotspot_name VARCHAR(100),
    dns_name VARCHAR(100),
    currency VARCHAR(10) DEFAULT 'Rp',
    phone VARCHAR(20),
    email VARCHAR(100),
    info_lp VARCHAR(255),
    idle_timeout VARCHAR(10) DEFAULT '30',
    report_mode VARCHAR(20) DEFAULT 'disable',
    token VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS voucher_sales (
    id BIGSERIAL PRIMARY KEY,
    sold_at TIMESTAMPTZ NOT NULL,
    username VARCHAR(100) NOT NULL,
    profile_name VARCHAR(100) NOT NULL,
    price BIGINT NOT NULL DEFAULT 0,
    ip_address VARCHAR(45),
    mac_address VARCHAR(17),
    validity VARCHAR(20),
    router_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_voucher_sales_router FOREIGN KEY (router_id) REFERENCES routers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS profile_price_mappings (
    id BIGSERIAL PRIMARY KEY,
    profile_name VARCHAR(100) NOT NULL,
    price BIGINT NOT NULL DEFAULT 0,
    selling_price BIGINT NOT NULL DEFAULT 0,
    validity VARCHAR(20),
    exp_mode VARCHAR(10),
    lock_user BOOLEAN DEFAULT FALSE,
    lock_server BOOLEAN DEFAULT FALSE,
    router_id BIGINT NOT NULL,
    CONSTRAINT fk_profile_price_router FOREIGN KEY (router_id) REFERENCES routers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS system_users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'admin',
    active BOOLEAN DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT,
    action VARCHAR(50) NOT NULL,
    entity VARCHAR(50) NOT NULL,
    entity_id VARCHAR(50),
    details TEXT,
    ip_address VARCHAR(45),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS print_templates (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_routers_deleted_at ON routers(deleted_at);
CREATE INDEX IF NOT EXISTS idx_voucher_sales_sold_at ON voucher_sales(sold_at);
CREATE INDEX IF NOT EXISTS idx_voucher_sales_router_id ON voucher_sales(router_id);
CREATE INDEX IF NOT EXISTS idx_voucher_sales_username ON voucher_sales(username);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_profile_price_mappings_router_id ON profile_price_mappings(router_id);
CREATE INDEX IF NOT EXISTS idx_system_users_deleted_at ON system_users(deleted_at);
