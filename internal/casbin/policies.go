package casbinx

// getRolePolicies returns the canonical role policy table from the planning document.
// Each row: [sub, dom, obj, act]. dom="*" means policy applies to all tenants.
func getRolePolicies() [][]string {
	return [][]string{
		// ════════════════════════════════════════
		// SUPERADMIN — akses penuh semua tenant
		// ════════════════════════════════════════
		{"superadmin", "*", "/api/v1/*", "*"},

		// ════════════════════════════════════════
		// OWNER — full access dalam tenant sendiri
		// ════════════════════════════════════════
		{"owner", "*", "/api/v1/tenant", "*"},
		{"owner", "*", "/api/v1/tenant/*", "*"},
		{"owner", "*", "/api/v1/users", "*"},
		{"owner", "*", "/api/v1/users/:id", "*"},
		{"owner", "*", "/api/v1/routers", "*"},
		{"owner", "*", "/api/v1/routers/:id", "*"},
		{"owner", "*", "/api/v1/routers/:id/*", "*"},
		{"owner", "*", "/api/v1/templates", "*"},
		{"owner", "*", "/api/v1/templates/*", "*"},

		// ════════════════════════════════════════
		// ADMIN — semua kecuali tenant settings dan billing
		// ════════════════════════════════════════
		{"admin", "*", "/api/v1/users", "GET"},
		{"admin", "*", "/api/v1/users", "POST"},
		{"admin", "*", "/api/v1/users/:id", "GET"},
		{"admin", "*", "/api/v1/users/:id", "PUT"},
		{"admin", "*", "/api/v1/users/:id", "DELETE"},
		{"admin", "*", "/api/v1/routers", "*"},
		{"admin", "*", "/api/v1/routers/:id", "*"},
		{"admin", "*", "/api/v1/routers/:id/*", "*"},
		{"admin", "*", "/api/v1/templates", "*"},
		{"admin", "*", "/api/v1/templates/*", "*"},

		// ════════════════════════════════════════
		// STAFF — operasional harian saja
		// ════════════════════════════════════════
		{"staff", "*", "/api/v1/routers", "GET"},
		{"staff", "*", "/api/v1/routers/:id", "GET"},
		{"staff", "*", "/api/v1/routers/:id/hotspot/users", "*"},
		{"staff", "*", "/api/v1/routers/:id/hotspot/users/:uid", "*"},
		{"staff", "*", "/api/v1/routers/:id/hotspot/active", "*"},
		{"staff", "*", "/api/v1/routers/:id/hotspot/inactive", "GET"},
		{"staff", "*", "/api/v1/routers/:id/hotspot/profiles", "GET"},
		{"staff", "*", "/api/v1/routers/:id/hotspot/servers", "GET"},
		{"staff", "*", "/api/v1/routers/:id/vouchers/*", "*"},
		{"staff", "*", "/api/v1/routers/:id/reports/*", "GET"},
		{"staff", "*", "/api/v1/templates", "GET"},
		{"staff", "*", "/api/v1/templates/:id", "GET"},
	}
}
