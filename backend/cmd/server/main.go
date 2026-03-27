package main

import (
	"log"
	"os"

	"github.com/yield-guard/backend/internal/api"
	"github.com/yield-guard/backend/internal/mlit"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mlitClient := mlit.NewClient()
	handler := api.NewHandler(mlitClient)
	router := api.NewRouter(handler)

	log.Printf("Yield-Guard backend starting on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
