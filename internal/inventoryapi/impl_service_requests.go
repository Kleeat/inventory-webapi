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

type implServiceRequestsAPI struct{}

func NewServiceRequestsAPI() ServiceRequestsAPI {
	return &implServiceRequestsAPI{}
}

func trimServiceRequestFields(title, description string) (string, string) {
	return strings.TrimSpace(title), strings.TrimSpace(description)
}

func validateServiceRequestCreate(title, description string, priority Priority, equipmentId string) error {
	if title == "" || description == "" || string(priority) == "" || equipmentId == "" {
		return fmt.Errorf("missing required fields: title, description, priority, equipmentId")
	}
	return nil
}

// validTransitions defines the allowed status state machine.
var validTransitions = map[ServiceRequestStatus][]ServiceRequestStatus{
	NEW:         {ASSIGNED},
	ASSIGNED:    {IN_PROGRESS, NEW},
	IN_PROGRESS: {CLOSED},
}

func isValidTransition(from, to ServiceRequestStatus) bool {
	for _, allowed := range validTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

func srLookupPipeline(matchStage bson.D, equipmentCollectionName string) bson.A {
	return bson.A{
		bson.D{{Key: "$match", Value: matchStage}},
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: equipmentCollectionName},
			{Key: "localField", Value: "equipmentId"},
			{Key: "foreignField", Value: "id"},
			{Key: "as", Value: "equipmentArr"},
		}}},
		bson.D{{Key: "$addFields", Value: bson.D{
			{Key: "equipment", Value: bson.D{{Key: "$arrayElemAt", Value: bson.A{"$equipmentArr", 0}}}},
		}}},
		bson.D{{Key: "$project", Value: bson.D{{Key: "equipmentArr", Value: 0}}}},
	}
}

type srAggResult struct {
	ServiceRequestDoc `bson:",inline"`
	EquipmentDoc      *EquipmentDoc `bson:"equipment"`
}

func decodeSrAggRow(raw bson.Raw) (*ServiceRequest, error) {
	var row srAggResult
	if err := bson.Unmarshal(raw, &row); err != nil {
		return nil, err
	}
	sr := serviceRequestFromDoc(&row.ServiceRequestDoc)
	if row.EquipmentDoc != nil {
		sr.Equipment = *equipmentFromDoc(row.EquipmentDoc)
	}
	return sr, nil
}

func (api *implServiceRequestsAPI) CreateServiceRequest(c *gin.Context) {
	srDb, ok := getServiceRequestDb(c)
	if !ok {
		return
	}
	equipmentDb, ok := getEquipmentDb(c)
	if !ok {
		return
	}

	var body ServiceRequestCreate
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	body.Title, body.Description = trimServiceRequestFields(body.Title, body.Description)
	body.EquipmentId = strings.TrimSpace(body.EquipmentId)

	if err := validateServiceRequestCreate(body.Title, body.Description, body.Priority, body.EquipmentId); err != nil {
		respondError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	eq, err := equipmentDb.FindDocument(ctx, body.EquipmentId)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Equipment not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to verify equipment", err)
		return
	}

	now := time.Now().UTC()
	doc := ServiceRequestDoc{
		Id:          uuid.New().String(),
		Title:       body.Title,
		Description: body.Description,
		Priority:    body.Priority,
		EquipmentId: body.EquipmentId,
		ReportedBy:  body.ReportedBy,
		Status:      NEW,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err = srDb.CreateDocument(ctx, doc.Id, &doc)
	switch err {
	case nil:
		sr := serviceRequestFromDoc(&doc)
		sr.Equipment = *equipmentFromDoc(eq)
		c.JSON(http.StatusCreated, sr)
	case db_service.ErrConflict:
		respondError(c, http.StatusConflict, "Service request already exists", err)
	default:
		respondError(c, http.StatusInternalServerError, "Failed to create service request", err)
	}
}

func (api *implServiceRequestsAPI) GetServiceRequest(c *gin.Context) {
	srDb, ok := getServiceRequestDb(c)
	if !ok {
		return
	}
	equipmentDb, ok := getEquipmentDb(c)
	if !ok {
		return
	}

	requestId := c.Param("requestId")

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	pipeline := srLookupPipeline(
		bson.D{{Key: "id", Value: requestId}},
		equipmentDb.CollectionName(),
	)

	rows, err := srDb.Aggregate(ctx, pipeline)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to retrieve service request", err)
		return
	}
	if len(rows) == 0 {
		respondError(c, http.StatusNotFound, "Service request not found", nil)
		return
	}

	sr, err := decodeSrAggRow(rows[0])
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to decode service request", err)
		return
	}
	c.JSON(http.StatusOK, sr)
}

func (api *implServiceRequestsAPI) ListServiceRequests(c *gin.Context) {
	srDb, ok := getServiceRequestDb(c)
	if !ok {
		return
	}
	equipmentDb, ok := getEquipmentDb(c)
	if !ok {
		return
	}

	page, pageSize := parsePage(c)

	filter := bson.D{}
	if v := c.Query("status"); v != "" {
		filter = append(filter, bson.E{Key: "status", Value: v})
	}
	if v := c.Query("equipmentId"); v != "" {
		filter = append(filter, bson.E{Key: "equipmentId", Value: v})
	}
	if v := c.Query("assignedTo"); v != "" {
		filter = append(filter, bson.E{Key: "assignedTo", Value: v})
	}
	if v := c.Query("priority"); v != "" {
		filter = append(filter, bson.E{Key: "priority", Value: v})
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	total, err := srDb.CountDocumentsByFilter(ctx, filter)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to count service requests", err)
		return
	}

	skip := int64((page - 1) * pageSize)
	pipeline := append(
		bson.A{
			bson.D{{Key: "$match", Value: filter}},
			bson.D{{Key: "$skip", Value: skip}},
			bson.D{{Key: "$limit", Value: int64(pageSize)}},
		},
		srLookupPipeline(bson.D{}, equipmentDb.CollectionName())[1:]...,
	)

	rows, err := srDb.Aggregate(ctx, pipeline)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list service requests", err)
		return
	}

	content := make([]ServiceRequest, len(rows))
	for i, raw := range rows {
		sr, err := decodeSrAggRow(raw)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "Failed to decode service request", err)
			return
		}
		content[i] = *sr
	}

	totalElements := int32(total)
	totalPages := (totalElements + int32(pageSize) - 1) / int32(pageSize)
	if totalPages == 0 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, ServiceRequestPage{
		Content:       content,
		TotalElements: totalElements,
		TotalPages:    totalPages,
		Page:          int32(page),
		PageSize:      int32(pageSize),
	})
}

func (api *implServiceRequestsAPI) UpdateServiceRequest(c *gin.Context) {
	srDb, ok := getServiceRequestDb(c)
	if !ok {
		return
	}
	equipmentDb, ok := getEquipmentDb(c)
	if !ok {
		return
	}

	requestId := c.Param("requestId")

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	existing, err := srDb.FindDocument(ctx, requestId)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Service request not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to retrieve service request", err)
		return
	}

	if existing.Status == CLOSED {
		respondError(c, http.StatusConflict, "Cannot edit a closed service request", nil)
		return
	}

	var body ServiceRequestUpdate
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	body.Title, body.Description = trimServiceRequestFields(body.Title, body.Description)

	updated := *existing
	if body.Title != "" {
		updated.Title = body.Title
	}
	if body.Description != "" {
		updated.Description = body.Description
	}
	if body.Priority != "" {
		updated.Priority = body.Priority
	}
	updated.UpdatedAt = time.Now().UTC()

	err = srDb.UpdateDocument(ctx, requestId, &updated)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Service request not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to update service request", err)
		return
	}

	sr := serviceRequestFromDoc(&updated)
	if eq, err := equipmentDb.FindDocument(ctx, updated.EquipmentId); err == nil {
		sr.Equipment = *equipmentFromDoc(eq)
	}
	c.JSON(http.StatusOK, sr)
}

func (api *implServiceRequestsAPI) DeleteServiceRequest(c *gin.Context) {
	srDb, ok := getServiceRequestDb(c)
	if !ok {
		return
	}

	requestId := c.Param("requestId")

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	existing, err := srDb.FindDocument(ctx, requestId)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Service request not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to retrieve service request", err)
		return
	}

	if existing.Status != NEW {
		respondError(c, http.StatusConflict, "Cannot delete a request that is already assigned or in progress", nil)
		return
	}

	err = srDb.DeleteDocument(ctx, requestId)
	switch err {
	case nil:
		c.Status(http.StatusNoContent)
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Service request not found", err)
	default:
		respondError(c, http.StatusInternalServerError, "Failed to delete service request", err)
	}
}

func (api *implServiceRequestsAPI) TransitionServiceRequestStatus(c *gin.Context) {
	srDb, ok := getServiceRequestDb(c)
	if !ok {
		return
	}
	equipmentDb, ok := getEquipmentDb(c)
	if !ok {
		return
	}

	requestId := c.Param("requestId")

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	existing, err := srDb.FindDocument(ctx, requestId)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Service request not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to retrieve service request", err)
		return
	}

	var body TransitionServiceRequestStatusRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if !isValidTransition(existing.Status, body.Status) {
		respondError(c, http.StatusUnprocessableEntity,
			fmt.Sprintf("Invalid status transition: %s → %s", existing.Status, body.Status), nil)
		return
	}

	if body.Status == ASSIGNED && strings.TrimSpace(body.AssignedTo) == "" {
		respondError(c, http.StatusBadRequest, "assignedTo is required when transitioning to ASSIGNED", nil)
		return
	}
	if body.Status == CLOSED && strings.TrimSpace(body.ResolutionNote) == "" {
		respondError(c, http.StatusBadRequest, "resolutionNote is required when transitioning to CLOSED", nil)
		return
	}

	updated := *existing
	updated.Status = body.Status
	updated.UpdatedAt = time.Now().UTC()

	switch body.Status {
	case ASSIGNED:
		assignedTo := strings.TrimSpace(body.AssignedTo)
		updated.AssignedTo = &assignedTo
	case NEW:
		updated.AssignedTo = nil
	case CLOSED:
		resolutionNote := strings.TrimSpace(body.ResolutionNote)
		updated.ResolutionNote = &resolutionNote
		now := time.Now().UTC()
		updated.ClosedAt = &now
	}

	err = srDb.UpdateDocument(ctx, requestId, &updated)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		respondError(c, http.StatusNotFound, "Service request not found", err)
		return
	default:
		respondError(c, http.StatusInternalServerError, "Failed to update service request status", err)
		return
	}

	sr := serviceRequestFromDoc(&updated)
	if eq, err := equipmentDb.FindDocument(ctx, updated.EquipmentId); err == nil {
		sr.Equipment = *equipmentFromDoc(eq)
	}
	c.JSON(http.StatusOK, sr)
}
