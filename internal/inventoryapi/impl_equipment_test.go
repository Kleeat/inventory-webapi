package inventoryapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Kleeat/inventory-webapi/internal/db_service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
)

type EquipmentAPISuite struct {
	suite.Suite
	equipmentDbMock      *DbServiceMock[EquipmentDoc]
	locationDbMock       *DbServiceMock[Location]
	serviceRequestDbMock *DbServiceMock[ServiceRequestDoc]
	sut                  EquipmentAPI
}

func TestEquipmentAPISuite(t *testing.T) {
	suite.Run(t, new(EquipmentAPISuite))
}

func (suite *EquipmentAPISuite) SetupTest() {
	suite.equipmentDbMock = &DbServiceMock[EquipmentDoc]{}
	suite.locationDbMock = &DbServiceMock[Location]{}
	suite.serviceRequestDbMock = &DbServiceMock[ServiceRequestDoc]{}

	var _ db_service.DbService[EquipmentDoc] = suite.equipmentDbMock
	var _ db_service.DbService[Location] = suite.locationDbMock
	var _ db_service.DbService[ServiceRequestDoc] = suite.serviceRequestDbMock

	suite.sut = NewEquipmentAPI()
}

func (suite *EquipmentAPISuite) makeCtx(method, target, body string, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set("db_equipment", suite.equipmentDbMock)
	ctx.Set("db_location", suite.locationDbMock)
	ctx.Set("db_service_request", suite.serviceRequestDbMock)
	ctx.Params = params
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	ctx.Request = req
	return ctx, rec
}

func sampleEquipmentDoc() *EquipmentDoc {
	return &EquipmentDoc{
		Id:              "eq-1",
		Name:            "MRI Scanner",
		Type:            "Imaging",
		InventoryNumber: "INV-001",
		LocationId:      "test-location",
		Status:          ACTIVE,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
}

type equipAggResult struct {
	EquipmentDoc `bson:",inline"`
	Location     Location `bson:"location"`
}

func marshalEquipAgg(doc *EquipmentDoc, loc *Location) bson.Raw {
	row := equipAggResult{EquipmentDoc: *doc}
	if loc != nil {
		row.Location = *loc
	}
	raw, _ := bson.Marshal(row)
	return raw
}

// --- CreateEquipment ---

func (suite *EquipmentAPISuite) Test_CreateEquipment_ValidBody_Returns201() {
	// ARRANGE
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "test-location").
		Return(sampleLocation(), nil)
	suite.equipmentDbMock.
		On("CreateDocument", mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	body := `{"name":"MRI Scanner","type":"Imaging","inventoryNumber":"INV-001","locationId":"test-location"}`
	ctx, rec := suite.makeCtx("POST", "/v1/equipment", body, nil)

	// ACT
	suite.sut.CreateEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusCreated, rec.Code)
	suite.equipmentDbMock.AssertCalled(suite.T(), "CreateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *EquipmentAPISuite) Test_CreateEquipment_MissingRequiredField_Returns400() {
	// ARRANGE
	body := `{"name":"","type":"Imaging","inventoryNumber":"INV-001","locationId":"test-location"}`
	ctx, rec := suite.makeCtx("POST", "/v1/equipment", body, nil)

	// ACT
	suite.sut.CreateEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusBadRequest, rec.Code)
	suite.equipmentDbMock.AssertNotCalled(suite.T(), "CreateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *EquipmentAPISuite) Test_CreateEquipment_InvalidJson_Returns400() {
	// ARRANGE
	ctx, rec := suite.makeCtx("POST", "/v1/equipment", "not-valid-json{", nil)

	// ACT
	suite.sut.CreateEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusBadRequest, rec.Code)
}

func (suite *EquipmentAPISuite) Test_CreateEquipment_LocationNotFound_Returns400() {
	// ARRANGE
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "missing-location").
		Return(nil, db_service.ErrNotFound)

	body := `{"name":"MRI Scanner","type":"Imaging","inventoryNumber":"INV-001","locationId":"missing-location"}`
	ctx, rec := suite.makeCtx("POST", "/v1/equipment", body, nil)

	// ACT
	suite.sut.CreateEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusBadRequest, rec.Code)
	suite.equipmentDbMock.AssertNotCalled(suite.T(), "CreateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *EquipmentAPISuite) Test_CreateEquipment_DbConflict_Returns409() {
	// ARRANGE
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "test-location").
		Return(sampleLocation(), nil)
	suite.equipmentDbMock.
		On("CreateDocument", mock.Anything, mock.Anything, mock.Anything).
		Return(db_service.ErrConflict)

	body := `{"name":"MRI Scanner","type":"Imaging","inventoryNumber":"INV-001","locationId":"test-location"}`
	ctx, rec := suite.makeCtx("POST", "/v1/equipment", body, nil)

	// ACT
	suite.sut.CreateEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusConflict, rec.Code)
}

// --- GetEquipment ---

func (suite *EquipmentAPISuite) Test_GetEquipment_Found_Returns200() {
	// ARRANGE
	suite.locationDbMock.On("CollectionName").Return("locations")
	suite.equipmentDbMock.
		On("Aggregate", mock.Anything, mock.Anything).
		Return([]bson.Raw{marshalEquipAgg(sampleEquipmentDoc(), sampleLocation())}, nil)

	ctx, rec := suite.makeCtx("GET", "/v1/equipment/eq-1", "", gin.Params{
		{Key: "equipmentId", Value: "eq-1"},
	})

	// ACT
	suite.sut.GetEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusOK, rec.Code)
}

func (suite *EquipmentAPISuite) Test_GetEquipment_NotFound_Returns404() {
	// ARRANGE
	suite.locationDbMock.On("CollectionName").Return("locations")
	suite.equipmentDbMock.
		On("Aggregate", mock.Anything, mock.Anything).
		Return([]bson.Raw{}, nil)

	ctx, rec := suite.makeCtx("GET", "/v1/equipment/missing-id", "", gin.Params{
		{Key: "equipmentId", Value: "missing-id"},
	})

	// ACT
	suite.sut.GetEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusNotFound, rec.Code)
}

// --- ListEquipment ---

func (suite *EquipmentAPISuite) Test_ListEquipment_NoFilter_Returns200() {
	// ARRANGE
	suite.locationDbMock.On("CollectionName").Return("locations")
	suite.equipmentDbMock.
		On("CountDocumentsByFilter", mock.Anything, mock.Anything).
		Return(int64(1), nil)
	suite.equipmentDbMock.
		On("Aggregate", mock.Anything, mock.Anything).
		Return([]bson.Raw{marshalEquipAgg(sampleEquipmentDoc(), sampleLocation())}, nil)

	ctx, rec := suite.makeCtx("GET", "/v1/equipment", "", nil)

	// ACT
	suite.sut.ListEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusOK, rec.Code)
	suite.equipmentDbMock.AssertCalled(suite.T(), "CountDocumentsByFilter", mock.Anything, mock.Anything)
	suite.equipmentDbMock.AssertCalled(suite.T(), "Aggregate", mock.Anything, mock.Anything)
}

func (suite *EquipmentAPISuite) Test_ListEquipment_WithStatusFilter_PassesFilterToDb() {
	// ARRANGE
	suite.locationDbMock.On("CollectionName").Return("locations")
	suite.equipmentDbMock.
		On("CountDocumentsByFilter", mock.Anything, mock.Anything).
		Return(int64(0), nil)
	suite.equipmentDbMock.
		On("Aggregate", mock.Anything, mock.Anything).
		Return([]bson.Raw{}, nil)

	ctx, rec := suite.makeCtx("GET", "/v1/equipment?status=ACTIVE", "", nil)

	// ACT
	suite.sut.ListEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusOK, rec.Code)

	var filterArg bson.D
	for _, call := range suite.equipmentDbMock.Calls {
		if call.Method == "CountDocumentsByFilter" {
			filterArg = call.Arguments.Get(1).(bson.D)
			break
		}
	}
	suite.Require().NotEmpty(filterArg, "expected a non-empty filter to be passed to CountDocumentsByFilter")
	suite.Equal("status", filterArg[0].Key)
	suite.Equal("ACTIVE", filterArg[0].Value)
}

// --- UpdateEquipment ---

func (suite *EquipmentAPISuite) Test_UpdateEquipment_ValidBody_Returns200() {
	// ARRANGE
	suite.equipmentDbMock.
		On("FindDocument", mock.Anything, "eq-1").
		Return(sampleEquipmentDoc(), nil)
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "test-location").
		Return(sampleLocation(), nil)
	suite.equipmentDbMock.
		On("UpdateDocument", mock.Anything, "eq-1", mock.Anything).
		Return(nil)

	body := `{"name":"MRI Scanner","type":"Imaging","inventoryNumber":"INV-001","locationId":"test-location","status":"ACTIVE"}`
	ctx, rec := suite.makeCtx("PUT", "/v1/equipment/eq-1", body, gin.Params{
		{Key: "equipmentId", Value: "eq-1"},
	})

	// ACT
	suite.sut.UpdateEquipment(ctx)

	// ASSERT
	suite.equipmentDbMock.AssertCalled(suite.T(), "UpdateDocument", mock.Anything, "eq-1", mock.Anything)
	suite.Equal(http.StatusOK, rec.Code)
}

func (suite *EquipmentAPISuite) Test_UpdateEquipment_NotFound_Returns404() {
	// ARRANGE
	suite.equipmentDbMock.
		On("FindDocument", mock.Anything, "missing-id").
		Return(nil, db_service.ErrNotFound)

	body := `{"name":"MRI Scanner","type":"Imaging","inventoryNumber":"INV-001","locationId":"test-location","status":"ACTIVE"}`
	ctx, rec := suite.makeCtx("PUT", "/v1/equipment/missing-id", body, gin.Params{
		{Key: "equipmentId", Value: "missing-id"},
	})

	// ACT
	suite.sut.UpdateEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusNotFound, rec.Code)
	suite.equipmentDbMock.AssertNotCalled(suite.T(), "UpdateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *EquipmentAPISuite) Test_UpdateEquipment_MissingRequiredField_Returns400() {
	// ARRANGE
	suite.equipmentDbMock.
		On("FindDocument", mock.Anything, "eq-1").
		Return(sampleEquipmentDoc(), nil)

	body := `{"name":"","type":"Imaging","inventoryNumber":"INV-001","locationId":"test-location","status":"ACTIVE"}`
	ctx, rec := suite.makeCtx("PUT", "/v1/equipment/eq-1", body, gin.Params{
		{Key: "equipmentId", Value: "eq-1"},
	})

	// ACT
	suite.sut.UpdateEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusBadRequest, rec.Code)
	suite.equipmentDbMock.AssertNotCalled(suite.T(), "UpdateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *EquipmentAPISuite) Test_UpdateEquipment_LocationNotFound_Returns400() {
	// ARRANGE
	suite.equipmentDbMock.
		On("FindDocument", mock.Anything, "eq-1").
		Return(sampleEquipmentDoc(), nil)
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "missing-location").
		Return(nil, db_service.ErrNotFound)

	body := `{"name":"MRI Scanner","type":"Imaging","inventoryNumber":"INV-001","locationId":"missing-location","status":"ACTIVE"}`
	ctx, rec := suite.makeCtx("PUT", "/v1/equipment/eq-1", body, gin.Params{
		{Key: "equipmentId", Value: "eq-1"},
	})

	// ACT
	suite.sut.UpdateEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusBadRequest, rec.Code)
	suite.equipmentDbMock.AssertNotCalled(suite.T(), "UpdateDocument", mock.Anything, mock.Anything, mock.Anything)
}

// --- DecommissionEquipment ---

func (suite *EquipmentAPISuite) Test_DecommissionEquipment_NoOpenRequests_Returns204() {
	// ARRANGE
	suite.equipmentDbMock.
		On("FindDocument", mock.Anything, "eq-1").
		Return(sampleEquipmentDoc(), nil)
	suite.serviceRequestDbMock.
		On("CountDocumentsByFilter", mock.Anything, mock.Anything).
		Return(int64(0), nil)
	suite.equipmentDbMock.
		On("UpdateDocument", mock.Anything, "eq-1", mock.Anything).
		Return(nil)

	ctx, _ := suite.makeCtx("DELETE", "/v1/equipment/eq-1", "", gin.Params{
		{Key: "equipmentId", Value: "eq-1"},
	})

	// ACT
	suite.sut.DecommissionEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusNoContent, ctx.Writer.Status())
	suite.equipmentDbMock.AssertCalled(suite.T(), "UpdateDocument", mock.Anything, "eq-1", mock.Anything)
}

func (suite *EquipmentAPISuite) Test_DecommissionEquipment_HasOpenRequests_Returns409() {
	// ARRANGE
	suite.equipmentDbMock.
		On("FindDocument", mock.Anything, "eq-1").
		Return(sampleEquipmentDoc(), nil)
	suite.serviceRequestDbMock.
		On("CountDocumentsByFilter", mock.Anything, mock.Anything).
		Return(int64(2), nil)

	ctx, rec := suite.makeCtx("DELETE", "/v1/equipment/eq-1", "", gin.Params{
		{Key: "equipmentId", Value: "eq-1"},
	})

	// ACT
	suite.sut.DecommissionEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusConflict, rec.Code)
	suite.equipmentDbMock.AssertNotCalled(suite.T(), "UpdateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *EquipmentAPISuite) Test_DecommissionEquipment_NotFound_Returns404() {
	// ARRANGE
	suite.equipmentDbMock.
		On("FindDocument", mock.Anything, "missing-id").
		Return(nil, db_service.ErrNotFound)

	ctx, rec := suite.makeCtx("DELETE", "/v1/equipment/missing-id", "", gin.Params{
		{Key: "equipmentId", Value: "missing-id"},
	})

	// ACT
	suite.sut.DecommissionEquipment(ctx)

	// ASSERT
	suite.Equal(http.StatusNotFound, rec.Code)
}

