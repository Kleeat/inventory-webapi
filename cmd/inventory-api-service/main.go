package main

import (
	"log"
	"os"
	"strings"

	"context"
	"time"

	"github.com/Kleeat/inventory-webapi/internal/db_service"
	"github.com/gin-contrib/cors"

	"github.com/Kleeat/inventory-webapi/api"
	"github.com/Kleeat/inventory-webapi/internal/inventoryapi"
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

	corsMiddleware := cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "PUT", "POST", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{""},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
	engine.Use(corsMiddleware)

	dbService := db_service.NewMongoService[inventoryapi.Equipment](db_service.MongoServiceConfig{})
	defer dbService.Disconnect(context.Background())
	engine.Use(func(ctx *gin.Context) {
		ctx.Set("db_service", dbService)
		ctx.Next()
	})

	handleFunctions := &inventoryapi.ApiHandleFunctions{
		EquipmentAPI:       inventoryapi.NewEquipmentAPI(),
		LocationsAPI:       inventoryapi.NewLocationsAPI(),
		ServiceRequestsAPI: inventoryapi.NewServiceRequestsAPI(),
	}
	inventoryapi.NewRouterWithGinEngine(engine, *handleFunctions)

	engine.GET("/openapi", api.HandleOpenApi)
	engine.Run(":" + port)
}
