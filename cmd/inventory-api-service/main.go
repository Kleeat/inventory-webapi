package main

import (
	"log"
	"os"
	"strings"

	"github.com/Kleeat/inventory-webapi/api"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Printf("Server started")
	port := os.Getenv("INVENTORY_API_PORT")
	if port == "" {
		port = "8080"
	}
	environment := os.Getenv("INVENTORY_API_ENVIRONMENT")
	if !strings.EqualFold(environment, "production") {
		gin.SetMode(gin.DebugMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery())

	engine.GET("/openapi", api.HandleOpenApi)
	engine.Run(":" + port)
}
