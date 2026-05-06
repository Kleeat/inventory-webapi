package inventoryapi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Kleeat/inventory-webapi/internal/db_service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

type implLocationsAPI struct{}

func NewLocationsAPI() LocationsAPI {
	return &implLocationsAPI{}
}

func parsePage(c *gin.Context) (page, pageSize int) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err = strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if err != nil || pageSize < 1 {
		pageSize = 20
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func trimLocationFields(building, floor, department, room, description string) (string, string, string, string, string) {
	return strings.TrimSpace(building),
		strings.TrimSpace(floor),
		strings.TrimSpace(department),
		strings.TrimSpace(room),
		strings.TrimSpace(description)
}

func validateLocationFields(building, floor, department, room string) error {
	if building == "" || floor == "" || department == "" || room == "" {
		return fmt.Errorf("missing required fields: building, floor, department, room")
	}
	return nil
}

func (api *implLocationsAPI) CreateLocation(c *gin.Context) {
	db, ok := getLocationDb(c)
	if !ok {
		return
	}

	var body LocationCreate
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	body.Building, body.Floor, body.Department, body.Room, body.Description =
		trimLocationFields(body.Building, body.Floor, body.Department, body.Room, body.Description)

	if err := validateLocationFields(body.Building, body.Floor, body.Department, body.Room); err != nil {
		respondError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	now := time.Now().UTC()
	location := Location{
		Id:          uuid.New().String(),
		Building:    body.Building,
		Floor:       body.Floor,
		Department:  body.Department,
		Room:        body.Room,
		Description: body.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	err := db.CreateDocument(ctx, location.Id, &location)
	switch err {
	case nil:
		c.JSON(http.StatusCreated, location)
	case db_service.ErrConflict:
		respondError(c, http.StatusConflict, "Location already exists", err)
	default:
		respondError(c, http.StatusInternalServerError, "Failed to create location", err)
	}
}

func (api *implLocationsAPI) GetLocation(c *gin.Context) {
	db, ok := getLocationDb(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	locationId := c.Param("locationId")
	location, err := db.FindDocument(ctx, locationId)
	switch err {
	case nil:
		c.JSON(http.StatusOK, location)
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Location not found", err)
	default:
		respondError(c, http.StatusInternalServerError, "Failed to retrieve location", err)
	}
}

func (api *implLocationsAPI) ListLocations(c *gin.Context) {
	db, ok := getLocationDb(c)
	if !ok {
		return
	}

	page, pageSize := parsePage(c)

	filter := bson.D{}
	if v := c.Query("building"); v != "" {
		filter = append(filter, bson.E{Key: "building", Value: v})
	}
	if v := c.Query("floor"); v != "" {
		filter = append(filter, bson.E{Key: "floor", Value: v})
	}
	if v := c.Query("department"); v != "" {
		filter = append(filter, bson.E{Key: "department", Value: v})
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	total, err := db.CountDocumentsByFilter(ctx, filter)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to count locations", err)
		return
	}

	skip := int64((page - 1) * pageSize)
	results, err := db.FindDocumentsByFilterPaginated(ctx, filter, skip, int64(pageSize))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list locations", err)
		return
	}

	// LocationPage.Content is []Location (generated type); copy from []*Location is unavoidable.
	content := make([]Location, len(results))
	for i, loc := range results {
		content[i] = *loc
	}

	totalElements := int32(total)
	totalPages := (totalElements + int32(pageSize) - 1) / int32(pageSize)
	if totalPages == 0 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, LocationPage{
		Content:       content,
		TotalElements: totalElements,
		TotalPages:    totalPages,
		Page:          int32(page),
		PageSize:      int32(pageSize),
	})
}

func (api *implLocationsAPI) UpdateLocation(c *gin.Context) {
	db, ok := getLocationDb(c)
	if !ok {
		return
	}

	locationId := c.Param("locationId")

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	existing, err := db.FindDocument(ctx, locationId)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Location not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to retrieve location", err)
		return
	}

	var body LocationUpdate
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	body.Building, body.Floor, body.Department, body.Room, body.Description =
		trimLocationFields(body.Building, body.Floor, body.Department, body.Room, body.Description)

	if err := validateLocationFields(body.Building, body.Floor, body.Department, body.Room); err != nil {
		respondError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	updated := Location{
		Id:          locationId,
		Building:    body.Building,
		Floor:       body.Floor,
		Department:  body.Department,
		Room:        body.Room,
		Description: body.Description,
		CreatedAt:   existing.CreatedAt,
		UpdatedAt:   time.Now().UTC(),
	}

	err = db.UpdateDocument(ctx, locationId, &updated)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Location not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to update location", err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (api *implLocationsAPI) DeleteLocation(c *gin.Context) {
	db, ok := getLocationDb(c)
	if !ok {
		return
	}
	equipmentDb, ok := getEquipmentDb(c)
	if !ok {
		return
	}

	locationId := c.Param("locationId")

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	_, err := db.FindDocument(ctx, locationId)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Location not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to retrieve location", err)
		return
	}

	activeCount, err := equipmentDb.CountDocumentsByFilter(ctx, bson.D{
		{Key: "locationId", Value: locationId},
		{Key: "status", Value: bson.D{{Key: "$ne", Value: string(DECOMMISSIONED)}}},
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to check equipment at location", err)
		return
	}
	if activeCount > 0 {
		respondError(c, http.StatusConflict, "Cannot delete location with active equipment assigned to it", nil)
		return
	}

	err = db.DeleteDocument(ctx, locationId)
	switch err {
	case nil:
		c.Status(http.StatusNoContent)
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Location not found", err)
	default:
		respondError(c, http.StatusInternalServerError, "Failed to delete location", err)
	}
}

func (api *implLocationsAPI) ListEquipmentAtLocation(c *gin.Context) {
	locationDb, ok := getLocationDb(c)
	if !ok {
		return
	}
	equipmentDb, ok := getEquipmentDb(c)
	if !ok {
		return
	}

	locationId := c.Param("locationId")

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	location, err := locationDb.FindDocument(ctx, locationId)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Location not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to retrieve location", err)
		return
	}

	page, pageSize := parsePage(c)

	eqFilter := bson.D{{Key: "locationId", Value: locationId}}
	if v := c.Query("status"); v != "" {
		eqFilter = append(eqFilter, bson.E{Key: "status", Value: v})
	}

	total, err := equipmentDb.CountDocumentsByFilter(ctx, eqFilter)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to count equipment", err)
		return
	}

	skip := int64((page - 1) * pageSize)
	results, err := equipmentDb.FindDocumentsByFilterPaginated(ctx, eqFilter, skip, int64(pageSize))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list equipment", err)
		return
	}

	// EquipmentPage.Content is []Equipment (generated type); copy from []*EquipmentDoc is unavoidable.
	content := make([]Equipment, len(results))
	for i, doc := range results {
		eq := equipmentFromDoc(doc)
		eq.Location = *location
		content[i] = *eq
	}

	totalElements := int32(total)
	totalPages := (totalElements + int32(pageSize) - 1) / int32(pageSize)
	if totalPages == 0 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, EquipmentPage{
		Content:       content,
		TotalElements: totalElements,
		TotalPages:    totalPages,
		Page:          int32(page),
		PageSize:      int32(pageSize),
	})
}
