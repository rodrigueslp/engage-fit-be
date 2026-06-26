package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/reports"
	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

type ReportsHandler struct {
	eligibleStudents reports.EligibleStudentsReportUseCase
	pendingRewards   reports.PendingRewardsReportUseCase
	monthlyFrequency reports.MonthlyFrequencyReportUseCase
	exporter         services.ReportExporter
}

func NewReportsHandler(eligibleStudents reports.EligibleStudentsReportUseCase, pendingRewards reports.PendingRewardsReportUseCase, monthlyFrequency reports.MonthlyFrequencyReportUseCase, exporter services.ReportExporter) ReportsHandler {
	return ReportsHandler{eligibleStudents: eligibleStudents, pendingRewards: pendingRewards, monthlyFrequency: monthlyFrequency, exporter: exporter}
}

func (h ReportsHandler) EligibleStudents(c *gin.Context) {
	boxID, ok := reportBoxID(c)
	if !ok {
		return
	}

	rows, err := h.eligibleStudents.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}

	if wantsCSV(c) {
		headers, csvRows := reports.EligibleStudentsCSV(rows)
		h.writeCSV(c, "relatorio-elegiveis.csv", headers, csvRows)
		return
	}

	response := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		response = append(response, gin.H{
			"campaign_id":         string(row.CampaignID),
			"campaign_name":       row.CampaignName,
			"student_id":          string(row.StudentID),
			"student_name":        row.StudentName,
			"student_phone":       row.StudentPhone,
			"source":              string(row.Source),
			"current_checkins":    row.CurrentCheckins,
			"target_checkins":     row.TargetCheckins,
			"remaining_checkins":  row.RemainingCheckins,
			"progress_percentage": row.ProgressPercentage,
			"reward_name":         row.RewardName,
		})
	}
	c.JSON(http.StatusOK, response)
}

func (h ReportsHandler) PendingRewards(c *gin.Context) {
	boxID, ok := reportBoxID(c)
	if !ok {
		return
	}

	rows, err := h.pendingRewards.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}

	if wantsCSV(c) {
		headers, csvRows := reports.PendingRewardsCSV(rows)
		h.writeCSV(c, "relatorio-brindes-pendentes.csv", headers, csvRows)
		return
	}

	response := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		response = append(response, gin.H{
			"id":            string(row.ID),
			"campaign_id":   string(row.CampaignID),
			"campaign_name": row.CampaignName,
			"reward_id":     string(row.RewardID),
			"reward_name":   row.RewardName,
			"student_id":    string(row.StudentID),
			"student_name":  row.StudentName,
			"student_phone": row.StudentPhone,
			"delivered":     row.Delivered,
			"delivered_at":  formatReportTime(row.DeliveredAt),
		})
	}
	c.JSON(http.StatusOK, response)
}

func (h ReportsHandler) MonthlyFrequency(c *gin.Context) {
	boxID, ok := reportBoxID(c)
	if !ok {
		return
	}

	period, label, ok := reportPeriod(c)
	if !ok {
		return
	}

	rows, err := h.monthlyFrequency.Execute(c.Request.Context(), boxID, period)
	if err != nil {
		respondError(c, err)
		return
	}

	if wantsCSV(c) {
		headers, csvRows := reports.MonthlyFrequencyCSV(rows)
		h.writeCSV(c, fmt.Sprintf("relatorio-frequencia-%s.csv", label), headers, csvRows)
		return
	}

	response := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		response = append(response, gin.H{
			"student_id":    string(row.StudentID),
			"student_name":  row.StudentName,
			"student_phone": row.StudentPhone,
			"source":        string(row.Source),
			"checkins":      row.Checkins,
			"first_checkin": formatReportTime(row.FirstCheckin),
			"last_checkin":  formatReportTime(row.LastCheckin),
		})
	}
	c.JSON(http.StatusOK, response)
}

func (h ReportsHandler) writeCSV(c *gin.Context, filename string, headers []string, rows [][]string) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	if err := h.exporter.WriteCSV(c.Request.Context(), c.Writer, headers, rows); err != nil {
		respondError(c, err)
		return
	}
}

func reportBoxID(c *gin.Context) (domain.ID, bool) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return "", false
	}
	return boxID, true
}

func wantsCSV(c *gin.Context) bool {
	return c.Query("format") == "csv"
}

func reportPeriod(c *gin.Context) (domain.TimeRange, string, bool) {
	month := c.Query("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}

	start, err := time.Parse("2006-01", month)
	if err != nil {
		respondBadRequest(c)
		return domain.TimeRange{}, "", false
	}

	return domain.TimeRange{Start: start, End: start.AddDate(0, 1, -1)}, month, true
}

func formatReportTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02")
}
