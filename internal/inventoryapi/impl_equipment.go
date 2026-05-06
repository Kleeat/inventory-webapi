package inventoryapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Kleeat/inventory-webapi/internal/db_service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

type implEquipmentAPI struct{}

func NewEquipmentAPI() EquipmentAPI {
	return &implEquipmentAPI{}
}

func trimEquipmentFields(name, equipType, inventoryNumber string) (string, string, string) {
	return strings.TrimSpace(name), strings.TrimSpace(equipType), strings.TrimSpace(inventoryNumber)
}

func validateEquipmentFields(name, equipType, inventoryNumber, locationId string) error {
	if name == "" || equipType == "" || inventoryNumber == "" || locationId == "" {
		return fmt.Errorf("missing required fields: name, type, inventoryNumber, locationId")
	}
	return nil
}

func (api *implEquipmentAPI) CreateEquipment(c *gin.Context) {
	db, ok := getEquipmentDb(c)
	if !ok {
		return
	}
	locationDb, ok := getLocationDb(c)
	if !ok {
		return
	}

	var body EquipmentCreate
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	body.Name, body.Type, body.InventoryNumber = trimEquipmentFields(body.Name, body.Type, body.InventoryNumber)
	body.LocationId = strings.TrimSpace(body.LocationId)

	if err := validateEquipmentFields(body.Name, body.Type, body.InventoryNumber, body.LocationId); err != nil {
		respondError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	location, err := locationDb.FindDocument(ctx, body.LocationId)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusBadRequest, "Location not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to verify location", err)
		return
	}

	now := time.Now().UTC()
	doc := EquipmentDoc{
		Id:              uuid.New().String(),
		Name:            body.Name,
		Type:            body.Type,
		InventoryNumber: body.InventoryNumber,
		WarrantyExpiry:  body.WarrantyExpiry,
		LocationId:      body.LocationId,
		Notes:           body.Notes,
		Status:          ACTIVE,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err = db.CreateDocument(ctx, doc.Id, &doc)
	switch err {
	case nil:
		eq := equipmentFromDoc(&doc)
		eq.Location = *location
		c.JSON(http.StatusCreated, eq)
	case db_service.ErrConflict:
		respondError(c, http.StatusConflict, "Equipment already exists", err)
	default:
		respondError(c, http.StatusInternalServerError, "Failed to create equipment", err)
	}
}

func (api *implEquipmentAPI) GetEquipment(c *gin.Context) {
	db, ok := getEquipmentDb(c)
	if !ok {
		return
	}
	locationDb, ok := getLocationDb(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	equipmentId := c.Param("equipmentId")
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "id", Value: equipmentId}}}},
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: locationDb.CollectionName()},
			{Key: "localField", Value: "locationId"},
			{Key: "foreignField", Value: "id"},
			{Key: "as", Value: "locationArr"},
		}}},
		bson.D{{Key: "$addFields", Value: bson.D{
			{Key: "location", Value: bson.D{{Key: "$arrayElemAt", Value: bson.A{"$locationArr", 0}}}},
		}}},
		bson.D{{Key: "$project", Value: bson.D{{Key: "locationArr", Value: 0}}}},
	}

	rows, err := db.Aggregate(ctx, pipeline)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to retrieve equipment", err)
		return
	}
	if len(rows) == 0 {
		respondError(c, http.StatusNotFound, "Equipment not found", nil)
		return
	}

	type aggResult struct {
		EquipmentDoc `bson:",inline"`
		Location     Location `bson:"location"`
	}
	var row aggResult
	if err := bson.Unmarshal(rows[0], &row); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to decode equipment", err)
		return
	}

	eq := equipmentFromDoc(&row.EquipmentDoc)
	eq.Location = row.Location
	c.JSON(http.StatusOK, eq)
}

func (api *implEquipmentAPI) ListEquipment(c *gin.Context) {
	db, ok := getEquipmentDb(c)
	if !ok {
		return
	}
	locationDb, ok := getLocationDb(c)
	if !ok {
		return
	}

	page, pageSize := parsePage(c)

	filter := bson.D{}
	if v := c.Query("status"); v != "" {
		filter = append(filter, bson.E{Key: "status", Value: v})
	}
	if v := c.Query("type"); v != "" {
		filter = append(filter, bson.E{Key: "type", Value: v})
	}
	if v := c.Query("locationId"); v != "" {
		filter = append(filter, bson.E{Key: "locationId", Value: v})
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	total, err := db.CountDocumentsByFilter(ctx, filter)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to count equipment", err)
		return
	}

	skip := int64((page - 1) * pageSize)
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: filter}},
		bson.D{{Key: "$skip", Value: skip}},
		bson.D{{Key: "$limit", Value: int64(pageSize)}},
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: locationDb.CollectionName()},
			{Key: "localField", Value: "locationId"},
			{Key: "foreignField", Value: "id"},
			{Key: "as", Value: "locationArr"},
		}}},
		bson.D{{Key: "$addFields", Value: bson.D{
			{Key: "location", Value: bson.D{{Key: "$arrayElemAt", Value: bson.A{"$locationArr", 0}}}},
		}}},
		bson.D{{Key: "$project", Value: bson.D{{Key: "locationArr", Value: 0}}}},
	}

	rows, err := db.Aggregate(ctx, pipeline)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list equipment", err)
		return
	}

	type aggResult struct {
		EquipmentDoc `bson:",inline"`
		Location     Location `bson:"location"`
	}

	content := make([]Equipment, len(rows))
	for i, raw := range rows {
		var row aggResult
		if err := bson.Unmarshal(raw, &row); err != nil {
			respondError(c, http.StatusInternalServerError, "Failed to decode equipment", err)
			return
		}
		eq := equipmentFromDoc(&row.EquipmentDoc)
		eq.Location = row.Location
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

func (api *implEquipmentAPI) UpdateEquipment(c *gin.Context) {
	db, ok := getEquipmentDb(c)
	if !ok {
		return
	}
	locationDb, ok := getLocationDb(c)
	if !ok {
		return
	}

	equipmentId := c.Param("equipmentId")

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	existing, err := db.FindDocument(ctx, equipmentId)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Equipment not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to retrieve equipment", err)
		return
	}

	var body EquipmentUpdate
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	body.Name, body.Type, body.InventoryNumber = trimEquipmentFields(body.Name, body.Type, body.InventoryNumber)
	body.LocationId = strings.TrimSpace(body.LocationId)

	if err := validateEquipmentFields(body.Name, body.Type, body.InventoryNumber, body.LocationId); err != nil {
		respondError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	location, err := locationDb.FindDocument(ctx, body.LocationId)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusBadRequest, "Location not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to verify location", err)
		return
	}

	updated := EquipmentDoc{
		Id:                      equipmentId,
		Name:                    body.Name,
		Type:                    body.Type,
		InventoryNumber:         body.InventoryNumber,
		WarrantyExpiry:          body.WarrantyExpiry,
		LocationId:              body.LocationId,
		Notes:                   body.Notes,
		Status:                  body.Status,
		OpenServiceRequestCount: existing.OpenServiceRequestCount,
		CreatedAt:               existing.CreatedAt,
		UpdatedAt:               time.Now().UTC(),
	}

	err = db.UpdateDocument(ctx, equipmentId, &updated)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Equipment not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to update equipment", err)
		return
	}

	eq := equipmentFromDoc(&updated)
	eq.Location = *location
	c.JSON(http.StatusOK, eq)
}

func (api *implEquipmentAPI) DecommissionEquipment(c *gin.Context) {
	db, ok := getEquipmentDb(c)
	if !ok {
		return
	}
	serviceRequestDb, ok := getServiceRequestDb(c)
	if !ok {
		return
	}

	equipmentId := c.Param("equipmentId")

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	existing, err := db.FindDocument(ctx, equipmentId)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Equipment not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to retrieve equipment", err)
		return
	}

	openCount, err := serviceRequestDb.CountDocumentsByFilter(ctx, bson.D{
		{Key: "equipmentId", Value: equipmentId},
		{Key: "status", Value: bson.D{{Key: "$ne", Value: string(CLOSED)}}},
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to check open service requests", err)
		return
	}
	if openCount > 0 {
		respondError(c, http.StatusConflict, "Cannot decommission equipment with open service requests", nil)
		return
	}

	decommissioned := *existing
	decommissioned.Status = DECOMMISSIONED
	decommissioned.UpdatedAt = time.Now().UTC()

	err = db.UpdateDocument(ctx, equipmentId, &decommissioned)
	switch err {
	case nil:
		c.Status(http.StatusNoContent)
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Equipment not found", err)
	default:
		respondError(c, http.StatusInternalServerError, "Failed to decommission equipment", err)
	}
}

