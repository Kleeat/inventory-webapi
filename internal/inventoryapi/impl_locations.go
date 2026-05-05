package inventoryapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type implLocationsAPI struct {
}

func NewLocationsAPI() LocationsAPI {
	return &implLocationsAPI{}
}

func (api *implLocationsAPI) CreateLocation(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implLocationsAPI) GetLocation(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implLocationsAPI) ListLocations(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implLocationsAPI) UpdateLocation(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implLocationsAPI) DeleteLocation(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}

func (api *implLocationsAPI) ListEquipmentAtLocation(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotImplemented)
}
