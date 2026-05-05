package inventoryapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type implServiceRequestsAPI struct {
}

func NewServiceRequestsAPI() ServiceRequestsAPI {
	return &implServiceRequestsAPI{}
}

func (api *implServiceRequestsAPI) CreateServiceRequest(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implServiceRequestsAPI) GetServiceRequest(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implServiceRequestsAPI) DeleteServiceRequest(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implServiceRequestsAPI) ListEquipmentServiceRequests(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implServiceRequestsAPI) ListServiceRequests(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implServiceRequestsAPI) TransitionServiceRequestStatus(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implServiceRequestsAPI) UpdateServiceRequest(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}
