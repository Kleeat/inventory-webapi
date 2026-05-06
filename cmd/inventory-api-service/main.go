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

	port := getPort()
	setupGinMode()

	engine := setupRouter()
	setupCorsMiddleware(engine)

	defer setupDb(engine)()

	setupRoutes(engine)

	engine.Run(":" + port)
}

func getPort() string {
	port := os.Getenv("INVENTORY_API_PORT")
	if port == "" {
		return "8080"
	}
	return port
}

func setupGinMode() {
	environment := os.Getenv("INVENTORY_API_ENVIRONMENT")
	if !strings.EqualFold(environment, "production") {
		gin.SetMode(gin.DebugMode)
	}
}

func setupRouter() *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	return engine
}

func setupCorsMiddleware(engine *gin.Engine) {
	corsMiddleware := cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "PUT", "POST", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{""},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})

	engine.Use(corsMiddleware)
}

func setupDb(engine *gin.Engine) func() {
	dbEquipment := db_service.NewMongoService[inventoryapi.EquipmentDoc](db_service.MongoServiceConfig{
		Collection: "equipment",
	})
	dbLocation := db_service.NewMongoService[inventoryapi.Location](db_service.MongoServiceConfig{
		Collection: "locations",
	})
	dbServiceRequest := db_service.NewMongoService[inventoryapi.ServiceRequest](db_service.MongoServiceConfig{
		Collection: "service_requests",
	})

	engine.Use(func(ctx *gin.Context) {
		ctx.Set("db_equipment", dbEquipment)
		ctx.Set("db_location", dbLocation)
		ctx.Set("db_service_request", dbServiceRequest)
		ctx.Next()
	})

	return func() {
		dbEquipment.Disconnect(context.Background())
		dbLocation.Disconnect(context.Background())
		dbServiceRequest.Disconnect(context.Background())
	}
}

func setupRoutes(engine *gin.Engine) {
	handleFunctions := &inventoryapi.ApiHandleFunctions{
		EquipmentAPI:       inventoryapi.NewEquipmentAPI(),
		LocationsAPI:       inventoryapi.NewLocationsAPI(),
		ServiceRequestsAPI: inventoryapi.NewServiceRequestsAPI(),
	}

	inventoryapi.NewRouterWithGinEngine(engine, *handleFunctions)

	engine.GET("/openapi", api.HandleOpenApi)
}
