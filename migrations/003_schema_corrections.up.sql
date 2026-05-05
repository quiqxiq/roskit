ALTER TABLE voucher_sales ADD COLUMN IF NOT EXISTS server VARCHAR(100);
ALTER TABLE voucher_sales ADD COLUMN IF NOT EXISTS selling_price BIGINT NOT NULL DEFAULT 0;
ALTER TABLE voucher_sales ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(200);

CREATE UNIQUE INDEX IF NOT EXISTS idx_voucher_sales_idempotency_key ON voucher_sales(idempotency_key);
