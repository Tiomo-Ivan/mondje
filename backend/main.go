package main

import (
	"log"

	"mondje-backend/config"
	"mondje-backend/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	config.ConnectDB()

	r := gin.Default()

	// Endpoints d'importation
	r.POST("/api/import", handlers.UploadImport)
	r.GET("/api/imports/:id/errors", handlers.GetImportErrors)
	r.DELETE("/api/imports/:id", handlers.DeleteImport)

	// Endpoints de consultation & solde
	r.GET("/api/transactions", handlers.GetTransactions)
	r.GET("/api/agencies/:name/balance", handlers.GetAgencyBalance)

	// Endpoints d'exportation & métriques (Défis 1, 14 & 15)
	r.GET("/api/export", handlers.ExportTransactions)
	r.GET("/api/metrics", handlers.GetSystemMetrics)

	log.Println("Serveur backend démarré sur le port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Échec du démarrage du serveur : %v", err)
	}
}
