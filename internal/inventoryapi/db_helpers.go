package inventoryapi

import (
	"log"
	"net/http"
	"time"

	"github.com/Kleeat/inventory-webapi/internal/db_service"
	"github.com/gin-gonic/gin"
)

const (
	dbTimeout   = 15 * time.Second
	maxPageSize = 100
)

func respondError(c *gin.Context, status int, message string, err error) {
	if err != nil {
		log.Printf("[%d] %s: %v", status, message, err)
	}
	body := gin.H{
		"status":  http.StatusText(status),
		"message": message,
	}
	if err != nil && status < 500 {
		body["error"] = err.Error()
	}
	c.JSON(status, body)
}

func getEquipmentDb(c *gin.Context) (db_service.DbService[EquipmentDoc], bool) {
	value, exists := c.Get("db_equipment")
	if !exists {
		respondError(c, http.StatusInternalServerError, "equipment db not found", nil)
		return nil, false
	}
	db, ok := value.(db_service.DbService[EquipmentDoc])
	if !ok {
		respondError(c, http.StatusInternalServerError, "equipment db context is not of required type", nil)
		return nil, false
	}
	return db, true
}

func getLocationDb(c *gin.Context) (db_service.DbService[Location], bool) {
	value, exists := c.Get("db_location")
	if !exists {
		respondError(c, http.StatusInternalServerError, "location db not found", nil)
		return nil, false
	}
	db, ok := value.(db_service.DbService[Location])
	if !ok {
		respondError(c, http.StatusInternalServerError, "location db context is not of required type", nil)
		return nil, false
	}
	return db, true
}

func getServiceRequestDb(c *gin.Context) (db_service.DbService[ServiceRequestDoc], bool) {
	value, exists := c.Get("db_service_request")
	if !exists {
		respondError(c, http.StatusInternalServerError, "service request db not found", nil)
		return nil, false
	}
	db, ok := value.(db_service.DbService[ServiceRequestDoc])
	if !ok {
		respondError(c, http.StatusInternalServerError, "service request db context is not of required type", nil)
		return nil, false
	}
	return db, true
}
