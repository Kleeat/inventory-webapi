package main

import (
	"log"
	"os"
	"strings"

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

	handleFunctions := &inventoryapi.ApiHandleFunctions{
		EquipmentAPI:       inventoryapi.NewEquipmentAPI(),
		LocationsAPI:       inventoryapi.NewLocationsAPI(),
		ServiceRequestsAPI: inventoryapi.NewServiceRequestsAPI(),
	}
	inventoryapi.NewRouterWithGinEngine(engine, *handleFunctions)

	engine.GET("/openapi", api.HandleOpenApi)
	engine.Run(":" + port)
}
