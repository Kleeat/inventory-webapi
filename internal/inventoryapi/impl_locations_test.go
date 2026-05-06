package inventoryapi

import (
	"context"
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

type DbServiceMock[DocType interface{}] struct {
	mock.Mock
}

func (m *DbServiceMock[DocType]) CreateDocument(ctx context.Context, id string, document *DocType) error {
	args := m.Called(ctx, id, document)
	return args.Error(0)
}

func (m *DbServiceMock[DocType]) FindDocument(ctx context.Context, id string) (*DocType, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DocType), args.Error(1)
}

func (m *DbServiceMock[DocType]) FindDocuments(ctx context.Context) ([]*DocType, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DocType), args.Error(1)
}

func (m *DbServiceMock[DocType]) FindDocumentsByFilter(ctx context.Context, filter bson.D) ([]*DocType, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DocType), args.Error(1)
}

func (m *DbServiceMock[DocType]) FindDocumentsByFilterPaginated(ctx context.Context, filter bson.D, skip, limit int64) ([]*DocType, error) {
	args := m.Called(ctx, filter, skip, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DocType), args.Error(1)
}

func (m *DbServiceMock[DocType]) CountDocumentsByFilter(ctx context.Context, filter bson.D) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *DbServiceMock[DocType]) UpdateDocument(ctx context.Context, id string, document *DocType) error {
	args := m.Called(ctx, id, document)
	return args.Error(0)
}

func (m *DbServiceMock[DocType]) DeleteDocument(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *DbServiceMock[DocType]) Disconnect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type LocationsAPISuite struct {
	suite.Suite
	locationDbMock  *DbServiceMock[Location]
	equipmentDbMock *DbServiceMock[EquipmentDoc]
	sut             LocationsAPI
}

func TestLocationsAPISuite(t *testing.T) {
	suite.Run(t, new(LocationsAPISuite))
}

func (suite *LocationsAPISuite) SetupTest() {
	suite.locationDbMock = &DbServiceMock[Location]{}
	suite.equipmentDbMock = &DbServiceMock[EquipmentDoc]{}

	var _ db_service.DbService[Location] = suite.locationDbMock
	var _ db_service.DbService[EquipmentDoc] = suite.equipmentDbMock

	suite.sut = NewLocationsAPI()
}

func (suite *LocationsAPISuite) makeCtx(method, target, body string, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set("db_location", suite.locationDbMock)
	ctx.Set("db_equipment", suite.equipmentDbMock)
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

func sampleLocation() *Location {
	return &Location{
		Id:         "test-location",
		Building:   "Building A",
		Floor:      "1",
		Department: "Cardiology",
		Room:       "101",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
}

func (suite *LocationsAPISuite) Test_CreateLocation_ValidBody_DbCreateCalled() {
	// ARRANGE
	suite.locationDbMock.
		On("CreateDocument", mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	body := `{"building":"Building A","floor":"1","department":"Cardiology","room":"101"}`
	ctx, rec := suite.makeCtx("POST", "/v1/locations", body, nil)

	// ACT
	suite.sut.CreateLocation(ctx)

	// ASSERT
	suite.locationDbMock.AssertCalled(suite.T(), "CreateDocument", mock.Anything, mock.Anything, mock.Anything)
	suite.Equal(http.StatusCreated, rec.Code)
}

func (suite *LocationsAPISuite) Test_CreateLocation_MissingRequiredFields_Returns400() {
	// ARRANGE
	body := `{"building":"","floor":"1","department":"Cardiology","room":"101"}`
	ctx, rec := suite.makeCtx("POST", "/v1/locations", body, nil)

	// ACT
	suite.sut.CreateLocation(ctx)

	// ASSERT
	suite.Equal(http.StatusBadRequest, rec.Code)
	suite.locationDbMock.AssertNotCalled(suite.T(), "CreateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *LocationsAPISuite) Test_CreateLocation_InvalidJson_Returns400() {
	// ARRANGE
	ctx, rec := suite.makeCtx("POST", "/v1/locations", "not-valid-json{", nil)

	// ACT
	suite.sut.CreateLocation(ctx)

	// ASSERT
	suite.Equal(http.StatusBadRequest, rec.Code)
}

func (suite *LocationsAPISuite) Test_CreateLocation_DbConflict_Returns409() {
	// ARRANGE
	suite.locationDbMock.
		On("CreateDocument", mock.Anything, mock.Anything, mock.Anything).
		Return(db_service.ErrConflict)

	body := `{"building":"Building A","floor":"1","department":"Cardiology","room":"101"}`
	ctx, rec := suite.makeCtx("POST", "/v1/locations", body, nil)

	// ACT
	suite.sut.CreateLocation(ctx)

	// ASSERT
	suite.Equal(http.StatusConflict, rec.Code)
}

func (suite *LocationsAPISuite) Test_GetLocation_ExistingLocation_Returns200() {
	// ARRANGE
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "test-location").
		Return(sampleLocation(), nil)

	ctx, rec := suite.makeCtx("GET", "/v1/locations/test-location", "", gin.Params{
		{Key: "locationId", Value: "test-location"},
	})

	// ACT
	suite.sut.GetLocation(ctx)

	// ASSERT
	suite.locationDbMock.AssertCalled(suite.T(), "FindDocument", mock.Anything, "test-location")
	suite.Equal(http.StatusOK, rec.Code)
}

func (suite *LocationsAPISuite) Test_GetLocation_NotFound_Returns404() {
	// ARRANGE
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "missing-id").
		Return(nil, db_service.ErrNotFound)

	ctx, rec := suite.makeCtx("GET", "/v1/locations/missing-id", "", gin.Params{
		{Key: "locationId", Value: "missing-id"},
	})

	// ACT
	suite.sut.GetLocation(ctx)

	// ASSERT
	suite.Equal(http.StatusNotFound, rec.Code)
}

func (suite *LocationsAPISuite) Test_ListLocations_NoFilter_ReturnsLocationPage() {
	// ARRANGE
	suite.locationDbMock.
		On("CountDocumentsByFilter", mock.Anything, mock.Anything).
		Return(int64(1), nil)
	suite.locationDbMock.
		On("FindDocumentsByFilterPaginated", mock.Anything, mock.Anything, int64(0), int64(20)).
		Return([]*Location{sampleLocation()}, nil)

	ctx, rec := suite.makeCtx("GET", "/v1/locations", "", nil)

	// ACT
	suite.sut.ListLocations(ctx)

	// ASSERT
	suite.Equal(http.StatusOK, rec.Code)
	suite.locationDbMock.AssertCalled(suite.T(), "CountDocumentsByFilter", mock.Anything, mock.Anything)
	suite.locationDbMock.AssertCalled(suite.T(), "FindDocumentsByFilterPaginated", mock.Anything, mock.Anything, int64(0), int64(20))
}

func (suite *LocationsAPISuite) Test_ListLocations_WithBuildingFilter_PassesFilterToDb() {
	// ARRANGE
	suite.locationDbMock.
		On("CountDocumentsByFilter", mock.Anything, mock.Anything).
		Return(int64(1), nil)
	suite.locationDbMock.
		On("FindDocumentsByFilterPaginated", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]*Location{sampleLocation()}, nil)

	ctx, rec := suite.makeCtx("GET", "/v1/locations?building=Building+A", "", nil)

	// ACT
	suite.sut.ListLocations(ctx)

	// ASSERT
	suite.Equal(http.StatusOK, rec.Code)

	var filterArg bson.D
	for _, call := range suite.locationDbMock.Calls {
		if call.Method == "CountDocumentsByFilter" {
			filterArg = call.Arguments.Get(1).(bson.D)
			break
		}
	}
	suite.Require().NotEmpty(filterArg, "expected a non-empty filter to be passed to CountDocumentsByFilter")
	suite.Equal("building", filterArg[0].Key)
	suite.Equal("Building A", filterArg[0].Value)
}

func (suite *LocationsAPISuite) Test_UpdateLocation_ValidBody_DbUpdateCalled() {
	// ARRANGE
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "test-location").
		Return(sampleLocation(), nil)
	suite.locationDbMock.
		On("UpdateDocument", mock.Anything, "test-location", mock.Anything).
		Return(nil)

	body := `{"building":"Building B","floor":"2","department":"Neurology","room":"202"}`
	ctx, rec := suite.makeCtx("PUT", "/v1/locations/test-location", body, gin.Params{
		{Key: "locationId", Value: "test-location"},
	})

	// ACT
	suite.sut.UpdateLocation(ctx)

	// ASSERT
	suite.locationDbMock.AssertCalled(suite.T(), "UpdateDocument", mock.Anything, "test-location", mock.Anything)
	suite.Equal(http.StatusOK, rec.Code)
}

func (suite *LocationsAPISuite) Test_UpdateLocation_NotFound_Returns404() {
	// ARRANGE
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "missing-id").
		Return(nil, db_service.ErrNotFound)

	body := `{"building":"Building B","floor":"2","department":"Neurology","room":"202"}`
	ctx, rec := suite.makeCtx("PUT", "/v1/locations/missing-id", body, gin.Params{
		{Key: "locationId", Value: "missing-id"},
	})

	// ACT
	suite.sut.UpdateLocation(ctx)

	// ASSERT
	suite.Equal(http.StatusNotFound, rec.Code)
	suite.locationDbMock.AssertNotCalled(suite.T(), "UpdateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *LocationsAPISuite) Test_UpdateLocation_MissingRequiredFields_Returns400() {
	// ARRANGE
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "test-location").
		Return(sampleLocation(), nil)

	body := `{"building":"Building B","floor":"2","department":"Neurology","room":""}`
	ctx, rec := suite.makeCtx("PUT", "/v1/locations/test-location", body, gin.Params{
		{Key: "locationId", Value: "test-location"},
	})

	// ACT
	suite.sut.UpdateLocation(ctx)

	// ASSERT
	suite.Equal(http.StatusBadRequest, rec.Code)
	suite.locationDbMock.AssertNotCalled(suite.T(), "UpdateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *LocationsAPISuite) Test_DeleteLocation_NoActiveEquipment_Returns204() {
	// ARRANGE
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "test-location").
		Return(sampleLocation(), nil)
	suite.equipmentDbMock.
		On("CountDocumentsByFilter", mock.Anything, mock.Anything).
		Return(int64(0), nil)
	suite.locationDbMock.
		On("DeleteDocument", mock.Anything, "test-location").
		Return(nil)

	ctx, _ := suite.makeCtx("DELETE", "/v1/locations/test-location", "", gin.Params{
		{Key: "locationId", Value: "test-location"},
	})

	// ACT
	suite.sut.DeleteLocation(ctx)

	// ASSERT
	suite.locationDbMock.AssertCalled(suite.T(), "DeleteDocument", mock.Anything, "test-location")
	// c.Status() (no body) sets gin's internal status but does not flush it to the recorder;
	// use ctx.Writer.Status() to read the value gin actually set.
	suite.Equal(http.StatusNoContent, ctx.Writer.Status())
}

func (suite *LocationsAPISuite) Test_DeleteLocation_ActiveEquipmentExists_Returns409() {
	// ARRANGE
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "test-location").
		Return(sampleLocation(), nil)
	suite.equipmentDbMock.
		On("CountDocumentsByFilter", mock.Anything, mock.Anything).
		Return(int64(2), nil)

	ctx, rec := suite.makeCtx("DELETE", "/v1/locations/test-location", "", gin.Params{
		{Key: "locationId", Value: "test-location"},
	})

	// ACT
	suite.sut.DeleteLocation(ctx)

	// ASSERT
	suite.Equal(http.StatusConflict, rec.Code)
	suite.locationDbMock.AssertNotCalled(suite.T(), "DeleteDocument", mock.Anything, mock.Anything)
}

func (suite *LocationsAPISuite) Test_DeleteLocation_LocationNotFound_Returns404() {
	// ARRANGE
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "missing-id").
		Return(nil, db_service.ErrNotFound)

	ctx, rec := suite.makeCtx("DELETE", "/v1/locations/missing-id", "", gin.Params{
		{Key: "locationId", Value: "missing-id"},
	})

	// ACT
	suite.sut.DeleteLocation(ctx)

	// ASSERT
	suite.Equal(http.StatusNotFound, rec.Code)
}

func (suite *LocationsAPISuite) Test_ListEquipmentAtLocation_LocationNotFound_Returns404() {
	// ARRANGE
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "missing-id").
		Return(nil, db_service.ErrNotFound)

	ctx, rec := suite.makeCtx("GET", "/v1/locations/missing-id/equipment", "", gin.Params{
		{Key: "locationId", Value: "missing-id"},
	})

	// ACT
	suite.sut.ListEquipmentAtLocation(ctx)

	// ASSERT
	suite.Equal(http.StatusNotFound, rec.Code)
}

func (suite *LocationsAPISuite) Test_ListEquipmentAtLocation_ReturnsEquipmentPageWithLocation() {
	// ARRANGE
	loc := sampleLocation()
	suite.locationDbMock.
		On("FindDocument", mock.Anything, "test-location").
		Return(loc, nil)

	eqDoc := &EquipmentDoc{
		Id:              "eq-1",
		Name:            "MRI Scanner",
		Type:            "Imaging",
		InventoryNumber: "INV-001",
		LocationId:      "test-location",
		Status:          ACTIVE,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	suite.equipmentDbMock.
		On("CountDocumentsByFilter", mock.Anything, mock.Anything).
		Return(int64(1), nil)
	suite.equipmentDbMock.
		On("FindDocumentsByFilterPaginated", mock.Anything, mock.Anything, int64(0), int64(20)).
		Return([]*EquipmentDoc{eqDoc}, nil)

	ctx, rec := suite.makeCtx("GET", "/v1/locations/test-location/equipment", "", gin.Params{
		{Key: "locationId", Value: "test-location"},
	})

	// ACT
	suite.sut.ListEquipmentAtLocation(ctx)

	// ASSERT
	suite.Equal(http.StatusOK, rec.Code)
	suite.equipmentDbMock.AssertCalled(suite.T(), "FindDocumentsByFilterPaginated", mock.Anything, mock.Anything, int64(0), int64(20))
}
