package inventoryapi

import "time"

// EquipmentDoc is the MongoDB storage representation of an equipment item.
// Location is intentionally absent — it is resolved at query time via LocationId.
type EquipmentDoc struct {
	Id                      string          `bson:"id"`
	Name                    string          `bson:"name"`
	Type                    string          `bson:"type"`
	InventoryNumber         string          `bson:"inventoryNumber"`
	WarrantyExpiry          string          `bson:"warrantyExpiry,omitempty"`
	LocationId              string          `bson:"locationId"`
	Notes                   string          `bson:"notes,omitempty"`
	Status                  EquipmentStatus `bson:"status"`
	OpenServiceRequestCount int32           `bson:"openServiceRequestCount,omitempty"`
	CreatedAt               time.Time       `bson:"createdAt"`
	UpdatedAt               time.Time       `bson:"updatedAt"`
}

func equipmentFromDoc(doc *EquipmentDoc) *Equipment {
	return &Equipment{
		Id:                      doc.Id,
		Name:                    doc.Name,
		Type:                    doc.Type,
		InventoryNumber:         doc.InventoryNumber,
		WarrantyExpiry:          doc.WarrantyExpiry,
		LocationId:              doc.LocationId,
		Notes:                   doc.Notes,
		Status:                  doc.Status,
		OpenServiceRequestCount: doc.OpenServiceRequestCount,
		CreatedAt:               doc.CreatedAt,
		UpdatedAt:               doc.UpdatedAt,
	}
}

func equipmentToDoc(eq *Equipment) *EquipmentDoc {
	return &EquipmentDoc{
		Id:                      eq.Id,
		Name:                    eq.Name,
		Type:                    eq.Type,
		InventoryNumber:         eq.InventoryNumber,
		WarrantyExpiry:          eq.WarrantyExpiry,
		LocationId:              eq.LocationId,
		Notes:                   eq.Notes,
		Status:                  eq.Status,
		OpenServiceRequestCount: eq.OpenServiceRequestCount,
		CreatedAt:               eq.CreatedAt,
		UpdatedAt:               eq.UpdatedAt,
	}
}
