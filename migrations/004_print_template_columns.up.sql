ALTER TABLE print_templates ADD COLUMN IF NOT EXISTS router_id BIGINT REFERENCES routers(id) ON DELETE CASCADE;
ALTER TABLE print_templates ADD COLUMN IF NOT EXISTS part VARCHAR(10) CHECK (part IN ('header', 'row', 'footer'));

CREATE INDEX IF NOT EXISTS idx_print_templates_router_id ON print_templates(router_id);
CREATE INDEX IF NOT EXISTS idx_print_templates_type ON print_templates(type);
