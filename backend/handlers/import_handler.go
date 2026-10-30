package handlers

import (
	"net/http"
	"strings"

	"mondje-backend/config"
	"mondje-backend/models"
	"mondje-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func UploadImport(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, "Fichier manquant")
		return
	}

	agencyName := c.PostForm("agency_name")
	if agencyName == "" {
		c.String(http.StatusBadRequest, "Nom de l'agence manquant")
		return
	}

	src, err := file.Open()
	if err != nil {
		c.String(http.StatusInternalServerError, "Erreur d'ouverture du fichier")
		return
	}
	defer src.Close()

	imp, err := services.ProcessCSVStream(c.Request.Context(), src, file.Filename, agencyName)
	if err != nil {
		c.String(http.StatusInternalServerError, "Échec du traitement de l'import : %v", err)
		return
	}

	c.JSON(http.StatusOK, imp)
}

func GetImportErrors(c *gin.Context) {
	importID := c.Param("id")

	rows, err := config.DB.Query(c.Request.Context(), `
		SELECT id, import_id, line_number, raw_content, reason, created_at 
		FROM import_errors 
		WHERE import_id = $1 
		ORDER BY line_number ASC`, importID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des erreurs"})
		return
	}
	defer rows.Close()

	var errorsList []models.ImportError
	for rows.Next() {
		var errItem models.ImportError
		if err := rows.Scan(&errItem.ID, &errItem.ImportID, &errItem.LineNumber, &errItem.RawContent, &errItem.Reason, &errItem.CreatedAt); err == nil {
			errorsList = append(errorsList, errItem)
		}
	}

	if errorsList == nil {
		errorsList = []models.ImportError{}
	}

	c.JSON(http.StatusOK, errorsList)
}

func DeleteImport(c *gin.Context) {
	idStr := c.Param("id")
	importID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format d'UUID d'import invalide"})
		return
	}

	err = services.DeleteImportRecord(c.Request.Context(), importID)
	if err != nil {
		if strings.Contains(err.Error(), "IMPORT_NOT_FOUND") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Import non trouvé"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Import et toutes ses données associées ont été supprimés avec succès",
		"import_id": importID,
	})
}
