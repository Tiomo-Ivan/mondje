package handlers

import (
	"net/http"
	"strconv"

	"mondje-backend/services"

	"github.com/gin-gonic/gin"
)

func GetTransactions(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)

	filter := services.TransactionFilter{
		AgencyName: c.Query("agency_name"),
		Type:       c.Query("type"),
		Search:     c.Query("search"),
		Cursor:     c.Query("cursor"),
		Limit:      limit,
	}

	result, err := services.GetTransactions(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func GetAgencyBalance(c *gin.Context) {
	agencyName := c.Param("name")
	if agencyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nom de l'agence requis"})
		return
	}

	balance, err := services.GetAgencyBalance(c.Request.Context(), agencyName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, balance)
}
