package inventoryapi

import "time"


// ServiceRequestDoc is the MongoDB storage representation of a service request.
// Equipment is intentionally absent — it is resolved at query time via EquipmentId.
type ServiceRequestDoc struct {
	Id          string               `bson:"id"`
	Title       string               `bson:"title"`
	Description string               `bson:"description"`
	Priority    Priority             `bson:"priority"`
	EquipmentId string               `bson:"equipmentId"`
	Status      ServiceRequestStatus `bson:"status"`
	CreatedAt   time.Time            `bson:"createdAt"`
	UpdatedAt   time.Time            `bson:"updatedAt"`
}

func serviceRequestFromDoc(doc *ServiceRequestDoc) *ServiceRequest {
	return &ServiceRequest{
		Id:          doc.Id,
		Title:       doc.Title,
		Description: doc.Description,
		Priority:    doc.Priority,
		EquipmentId: doc.EquipmentId,
		Status:      doc.Status,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
}
