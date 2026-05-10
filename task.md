# Task: Simplifikasi Multi-Tenant → Single-Instance Multi-Router

## Keputusan yang Sudah Dikonfirmasi
- Role: `admin` + `staff` saja
- Casbin: dihapus, pakai role di JWT
- Data: fresh start (drop + recreate)
- Webhook token: reset, tidak masalah

---

## Layer 1: Models
- [x] DELETE `models/tenant.go`
- [x] DELETE `models/tenant_settings.go`
- [x] NEW `models/settings.go`
- [x] MODIFY `models/user.go` — hapus TenantID, hapus owner/superadmin
- [x] MODIFY `models/router.go` — hapus TenantID
- [x] MODIFY `models/voucher_sale.go` — hapus TenantID
- [x] MODIFY `models/audit_log.go` — hapus TenantID
- [x] MODIFY `models/print_template.go` — hapus TenantID

## Layer 2: Repository
- [x] DELETE `repository/tenant_repo.go`
- [x] DELETE `repository/tenant_settings_repo.go`
- [x] NEW `repository/settings_repo.go`
- [x] MODIFY `repository/user_repo.go` — hapus tenant scoping
- [x] MODIFY `repository/router_repo.go` — hapus tenantID params
- [x] MODIFY `repository/sale_repo.go` — hapus tenantID params
- [x] MODIFY `repository/template_repo.go` — hapus tenant filter

## Layer 3: Services
- [x] DELETE `services/tenant_service.go`
- [x] NEW `services/settings_service.go`
- [x] MODIFY `services/auth_service.go` — hapus tenant dari login/JWT
- [x] MODIFY `services/router_service.go` — hapus tenantID
- [x] MODIFY `services/hotspot_service.go` — hapus tenantID
- [x] MODIFY `services/voucher_service.go` — hapus tenantID
- [x] MODIFY `services/report_service.go` — hapus tenantID

## Layer 4: Casbin
- [x] DELETE seluruh package `internal/casbin/`

## Layer 5: Middleware
- [x] DELETE `middleware/tenant.go`
- [x] DELETE `middleware/casbin.go`
- [x] NEW `middleware/role.go`
- [x] MODIFY `middleware/auth.go` — hapus tenant dari context
- [x] MODIFY `middleware/audit.go` — hapus tenantID

## Layer 6: Handlers
- [x] DELETE `handlers/tenant_handler.go`
- [x] NEW `handlers/settings_handler.go`
- [x] MODIFY `handlers/auth_handler.go` — simplifikasi login + setup
- [x] MODIFY `handlers/user_handler.go` — hapus tenant scope
- [x] MODIFY `handlers/router_handler.go` — hapus tenantID
- [x] MODIFY `handlers/hotspot_handler.go` — hapus tenantID
- [x] MODIFY `handlers/voucher_handler.go` — hapus tenantID
- [x] MODIFY `handlers/report_handler.go` — hapus tenantID
- [x] MODIFY `handlers/event_handler.go` — hapus tenantID
- [x] MODIFY `handlers/context.go` — hapus tenantIDFromCtx

## Layer 7: Router
- [x] MODIFY `api/router.go` — hapus Casbin/Tenant middleware, restructure routes

## Layer 8: DI Wiring
- [x] MODIFY `cmd/api/main.go` — hapus Casbin/Tenant wiring

## Layer 9: Verifikasi
- [ ] `go build ./...` harus pass
- [ ] Fix compile errors
