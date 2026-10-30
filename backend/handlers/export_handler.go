package handlers

import (
	"fmt"
	"net/http"
	"time"

	"mondje-backend/services"

	"github.com/gin-gonic/gin"
)

func ExportTransactions(c *gin.Context) {
	agencyName := c.Query("agency_name")

	fileName := fmt.Sprintf("export_transactions_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))

	err := services.ExportTransactionsCSV(c.Request.Context(), c.Writer, agencyName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

func GetSystemMetrics(c *gin.Context) {
	metrics, err := services.GetSystemMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}
