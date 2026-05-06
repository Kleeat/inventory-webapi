package inventoryapi

import "time"

// ServiceRequestDoc is the MongoDB storage representation of a service request.
// Equipment is intentionally absent — it is resolved at query time via EquipmentId.
type ServiceRequestDoc struct {
	Id             string               `bson:"id"`
	Title          string               `bson:"title"`
	Description    string               `bson:"description"`
	Priority       Priority             `bson:"priority"`
	EquipmentId    string               `bson:"equipmentId"`
	ReportedBy     string               `bson:"reportedBy,omitempty"`
	AssignedTo     *string              `bson:"assignedTo,omitempty"`
	ResolutionNote *string              `bson:"resolutionNote,omitempty"`
	Status         ServiceRequestStatus `bson:"status"`
	ClosedAt       *time.Time           `bson:"closedAt,omitempty"`
	CreatedAt      time.Time            `bson:"createdAt"`
	UpdatedAt      time.Time            `bson:"updatedAt"`
}

func serviceRequestFromDoc(doc *ServiceRequestDoc) *ServiceRequest {
	return &ServiceRequest{
		Id:             doc.Id,
		Title:          doc.Title,
		Description:    doc.Description,
		Priority:       doc.Priority,
		EquipmentId:    doc.EquipmentId,
		ReportedBy:     doc.ReportedBy,
		AssignedTo:     doc.AssignedTo,
		ResolutionNote: doc.ResolutionNote,
		Status:         doc.Status,
		ClosedAt:       doc.ClosedAt,
		CreatedAt:      doc.CreatedAt,
		UpdatedAt:      doc.UpdatedAt,
	}
}
