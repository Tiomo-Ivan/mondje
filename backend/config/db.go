package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func ConnectDB() *pgxpool.Pool {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://mondje_user:mondje_password@db:5432/mondje?sslmode=disable"
	}

	var pool *pgxpool.Pool
	var err error

	// Essayer jusqu'à 10 fois d'attendre PostgreSQL
	for i := 1; i <= 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool, err = pgxpool.New(ctx, dbURL)
		if err == nil {
			err = pool.Ping(ctx)
			if err == nil {
				cancel()
				fmt.Println("Connexion réussie à PostgreSQL !")
				DB = pool
				return pool
			}
		}
		cancel()
		log.Printf("Attente de PostgreSQL (tentative %d/10)...", i)
		time.Sleep(2 * time.Second)
	}

	log.Fatalf("Impossible de se connecter à PostgreSQL après 10 tentatives : %v", err)
	return nil
}