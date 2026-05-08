package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/services"
)

type ReportHandler struct {
	svc *services.ReportService
}

func NewReportHandler(svc *services.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

// resolveScope returns (tenantID, *routerID).
// If the route is /routers/:routerId/reports/* the routerID param is set.
// Otherwise (tenant-wide reports), routerID is nil.
func resolveScope(c *gin.Context) (uint, *uint, bool) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return 0, nil, false
	}
	if c.Param("routerId") == "" {
		return tenantID, nil, true
	}
	rid, err := strconv.ParseUint(c.Param("routerId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid router id"})
		return 0, nil, false
	}
	r := uint(rid)
	return tenantID, &r, true
}

func (h *ReportHandler) GetDailyReport(c *gin.Context) {
	tenantID, routerID, ok := resolveScope(c)
	if !ok {
		return
	}

	dateStr := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid date format, use YYYY-MM-DD"})
		return
	}

	filters := services.SaleFilters{
		Profile: c.Query("profile"),
		Server:  c.Query("server"),
		Search:  c.Query("search"),
	}

	result, err := h.svc.GetDailyReport(c.Request.Context(), tenantID, routerID, date, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get daily report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *ReportHandler) GetMonthlyReport(c *gin.Context) {
	tenantID, routerID, ok := resolveScope(c)
	if !ok {
		return
	}

	now := time.Now()
	year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(now.Year())))
	month, _ := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(now.Month()))))

	if month < 1 || month > 12 {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "month must be 1-12"})
		return
	}

	result, err := h.svc.GetMonthlyReport(c.Request.Context(), tenantID, routerID, year, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get monthly report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *ReportHandler) GetResumeReport(c *gin.Context) {
	tenantID, routerID, ok := resolveScope(c)
	if !ok {
		return
	}

	year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(time.Now().Year())))

	result, err := h.svc.GetResumeReport(c.Request.Context(), tenantID, routerID, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get resume report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *ReportHandler) GetDashboardSummary(c *gin.Context) {
	tenantID, routerID, ok := resolveScope(c)
	if !ok {
		return
	}

	result, err := h.svc.GetDashboardSummary(c.Request.Context(), tenantID, routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get dashboard summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *ReportHandler) GetDailySummary(c *gin.Context) {
	tenantID, routerID, ok := resolveScope(c)
	if !ok {
		return
	}

	monthStr := c.DefaultQuery("month", time.Now().Format("2006-01"))
	parsedMonth, err := time.Parse("2006-01", monthStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid month format, use YYYY-MM"})
		return
	}

	result, err := h.svc.GetResumeReport(c.Request.Context(), tenantID, routerID, parsedMonth.Year())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get daily summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *ReportHandler) ExportCSV(c *gin.Context) {
	tenantID, routerID, ok := resolveScope(c)
	if !ok {
		return
	}

	from, to, err := parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	filters := services.SaleFilters{
		Profile: c.Query("profile"),
		Server:  c.Query("server"),
	}

	data, err := h.svc.ExportCSV(c.Request.Context(), tenantID, routerID, from, to, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to export CSV"})
		return
	}

	filename := fmt.Sprintf("mikhmon-report-%s.csv", from.Format("2006-01"))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "text/csv", data)
}

func (h *ReportHandler) ExportExcel(c *gin.Context) {
	tenantID, routerID, ok := resolveScope(c)
	if !ok {
		return
	}

	from, to, err := parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	filters := services.SaleFilters{
		Profile: c.Query("profile"),
		Server:  c.Query("server"),
	}

	data, err := h.svc.ExportExcel(c.Request.Context(), tenantID, routerID, from, to, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to export Excel"})
		return
	}

	filename := fmt.Sprintf("mikhmon-report-%s.xlsx", from.Format("2006-01"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func parseDateRange(c *gin.Context) (time.Time, time.Time, error) {
	fromStr := c.DefaultQuery("from", time.Now().Format("2006-01-02"))
	toStr := c.DefaultQuery("to", time.Now().Format("2006-01-02"))

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid from date, use YYYY-MM-DD")
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid to date, use YYYY-MM-DD")
	}

	return from, to, nil
}
