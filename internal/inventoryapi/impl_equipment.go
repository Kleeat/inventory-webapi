package inventoryapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type implEquipmentAPI struct {
}

func NewEquipmentAPI() EquipmentAPI {
	return &implEquipmentAPI{}
}

func (api *implEquipmentAPI) CreateEquipment(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implEquipmentAPI) DecommissionEquipment(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implEquipmentAPI) GetEquipment(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implEquipmentAPI) ListEquipment(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implEquipmentAPI) ListEquipmentAtLocation(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implEquipmentAPI) ListEquipmentServiceRequests(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implEquipmentAPI) MoveEquipment(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implEquipmentAPI) UpdateEquipment(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}
