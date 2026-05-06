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

type ServiceRequestsAPISuite struct {
	suite.Suite
	srDbMock        *DbServiceMock[ServiceRequestDoc]
	equipmentDbMock *DbServiceMock[EquipmentDoc]
	sut             ServiceRequestsAPI
}

func TestServiceRequestsAPISuite(t *testing.T) {
	suite.Run(t, new(ServiceRequestsAPISuite))
}

func (suite *ServiceRequestsAPISuite) SetupTest() {
	suite.srDbMock = &DbServiceMock[ServiceRequestDoc]{}
	suite.equipmentDbMock = &DbServiceMock[EquipmentDoc]{}

	var _ db_service.DbService[ServiceRequestDoc] = suite.srDbMock
	var _ db_service.DbService[EquipmentDoc] = suite.equipmentDbMock

	suite.sut = NewServiceRequestsAPI()
}

func (suite *ServiceRequestsAPISuite) makeCtx(method, target, body string, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set("db_service_request", suite.srDbMock)
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

func sampleServiceRequestDoc() *ServiceRequestDoc {
	return &ServiceRequestDoc{
		Id:          "sr-1",
		Title:       "Display malfunction",
		Description: "Screen stops responding after 30 minutes.",
		Priority:    HIGH,
		EquipmentId: "eq-1",
		Status:      NEW,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

func marshalSrAgg(doc *ServiceRequestDoc, eqDoc *EquipmentDoc) bson.Raw {
	row := srAggResult{ServiceRequestDoc: *doc}
	if eqDoc != nil {
		row.EquipmentDoc = eqDoc
	}
	raw, _ := bson.Marshal(row)
	return raw
}

// --- CreateServiceRequest ---

func (suite *ServiceRequestsAPISuite) Test_CreateServiceRequest_ValidBody_Returns201() {
	// ARRANGE
	suite.equipmentDbMock.
		On("FindDocument", mock.Anything, "eq-1").
		Return(sampleEquipmentDoc(), nil)
	suite.srDbMock.
		On("CreateDocument", mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	body := `{"title":"Display malfunction","description":"Screen issue","priority":"HIGH","equipmentId":"eq-1"}`
	ctx, rec := suite.makeCtx("POST", "/v1/service-requests", body, nil)

	// ACT
	suite.sut.CreateServiceRequest(ctx)

	// ASSERT
	suite.Equal(http.StatusCreated, rec.Code)
	suite.srDbMock.AssertCalled(suite.T(), "CreateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *ServiceRequestsAPISuite) Test_CreateServiceRequest_MissingRequiredField_Returns400() {
	// ARRANGE
	body := `{"title":"","description":"Screen issue","priority":"HIGH","equipmentId":"eq-1"}`
	ctx, rec := suite.makeCtx("POST", "/v1/service-requests", body, nil)

	// ACT
	suite.sut.CreateServiceRequest(ctx)

	// ASSERT
	suite.Equal(http.StatusBadRequest, rec.Code)
	suite.srDbMock.AssertNotCalled(suite.T(), "CreateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *ServiceRequestsAPISuite) Test_CreateServiceRequest_InvalidJson_Returns400() {
	// ARRANGE
	ctx, rec := suite.makeCtx("POST", "/v1/service-requests", "not-valid-json{", nil)

	// ACT
	suite.sut.CreateServiceRequest(ctx)

	// ASSERT
	suite.Equal(http.StatusBadRequest, rec.Code)
}

func (suite *ServiceRequestsAPISuite) Test_CreateServiceRequest_EquipmentNotFound_Returns404() {
	// ARRANGE
	suite.equipmentDbMock.
		On("FindDocument", mock.Anything, "missing-eq").
		Return(nil, db_service.ErrNotFound)

	body := `{"title":"Display malfunction","description":"Screen issue","priority":"HIGH","equipmentId":"missing-eq"}`
	ctx, rec := suite.makeCtx("POST", "/v1/service-requests", body, nil)

	// ACT
	suite.sut.CreateServiceRequest(ctx)

	// ASSERT
	suite.Equal(http.StatusNotFound, rec.Code)
	suite.srDbMock.AssertNotCalled(suite.T(), "CreateDocument", mock.Anything, mock.Anything, mock.Anything)
}

// --- GetServiceRequest ---

func (suite *ServiceRequestsAPISuite) Test_GetServiceRequest_Found_Returns200() {
	// ARRANGE
	suite.equipmentDbMock.On("CollectionName").Return("equipment")
	suite.srDbMock.
		On("Aggregate", mock.Anything, mock.Anything).
		Return([]bson.Raw{marshalSrAgg(sampleServiceRequestDoc(), sampleEquipmentDoc())}, nil)

	ctx, rec := suite.makeCtx("GET", "/v1/service-requests/sr-1", "", gin.Params{
		{Key: "requestId", Value: "sr-1"},
	})

	// ACT
	suite.sut.GetServiceRequest(ctx)

	// ASSERT
	suite.Equal(http.StatusOK, rec.Code)
}

func (suite *ServiceRequestsAPISuite) Test_GetServiceRequest_NotFound_Returns404() {
	// ARRANGE
	suite.equipmentDbMock.On("CollectionName").Return("equipment")
	suite.srDbMock.
		On("Aggregate", mock.Anything, mock.Anything).
		Return([]bson.Raw{}, nil)

	ctx, rec := suite.makeCtx("GET", "/v1/service-requests/missing-id", "", gin.Params{
		{Key: "requestId", Value: "missing-id"},
	})

	// ACT
	suite.sut.GetServiceRequest(ctx)

	// ASSERT
	suite.Equal(http.StatusNotFound, rec.Code)
}

// --- ListServiceRequests ---

func (suite *ServiceRequestsAPISuite) Test_ListServiceRequests_NoFilter_Returns200() {
	// ARRANGE
	suite.equipmentDbMock.On("CollectionName").Return("equipment")
	suite.srDbMock.
		On("CountDocumentsByFilter", mock.Anything, mock.Anything).
		Return(int64(1), nil)
	suite.srDbMock.
		On("Aggregate", mock.Anything, mock.Anything).
		Return([]bson.Raw{marshalSrAgg(sampleServiceRequestDoc(), sampleEquipmentDoc())}, nil)

	ctx, rec := suite.makeCtx("GET", "/v1/service-requests", "", nil)

	// ACT
	suite.sut.ListServiceRequests(ctx)

	// ASSERT
	suite.Equal(http.StatusOK, rec.Code)
	suite.srDbMock.AssertCalled(suite.T(), "CountDocumentsByFilter", mock.Anything, mock.Anything)
	suite.srDbMock.AssertCalled(suite.T(), "Aggregate", mock.Anything, mock.Anything)
}

func (suite *ServiceRequestsAPISuite) Test_ListServiceRequests_WithStatusFilter_PassesFilterToDb() {
	// ARRANGE
	suite.equipmentDbMock.On("CollectionName").Return("equipment")
	suite.srDbMock.
		On("CountDocumentsByFilter", mock.Anything, mock.Anything).
		Return(int64(0), nil)
	suite.srDbMock.
		On("Aggregate", mock.Anything, mock.Anything).
		Return([]bson.Raw{}, nil)

	ctx, rec := suite.makeCtx("GET", "/v1/service-requests?status=NEW", "", nil)

	// ACT
	suite.sut.ListServiceRequests(ctx)

	// ASSERT
	suite.Equal(http.StatusOK, rec.Code)

	var filterArg bson.D
	for _, call := range suite.srDbMock.Calls {
		if call.Method == "CountDocumentsByFilter" {
			filterArg = call.Arguments.Get(1).(bson.D)
			break
		}
	}
	suite.Require().NotEmpty(filterArg, "expected a non-empty filter to be passed to CountDocumentsByFilter")
	suite.Equal("status", filterArg[0].Key)
	suite.Equal("NEW", filterArg[0].Value)
}

// --- UpdateServiceRequest ---

func (suite *ServiceRequestsAPISuite) Test_UpdateServiceRequest_ValidBody_Returns200() {
	// ARRANGE
	suite.srDbMock.
		On("FindDocument", mock.Anything, "sr-1").
		Return(sampleServiceRequestDoc(), nil)
	suite.srDbMock.
		On("UpdateDocument", mock.Anything, "sr-1", mock.Anything).
		Return(nil)
	suite.equipmentDbMock.
		On("FindDocument", mock.Anything, "eq-1").
		Return(sampleEquipmentDoc(), nil)

	body := `{"title":"Updated title","description":"Updated description","priority":"CRITICAL"}`
	ctx, rec := suite.makeCtx("PUT", "/v1/service-requests/sr-1", body, gin.Params{
		{Key: "requestId", Value: "sr-1"},
	})

	// ACT
	suite.sut.UpdateServiceRequest(ctx)

	// ASSERT
	suite.Equal(http.StatusOK, rec.Code)
	suite.srDbMock.AssertCalled(suite.T(), "UpdateDocument", mock.Anything, "sr-1", mock.Anything)
}

func (suite *ServiceRequestsAPISuite) Test_UpdateServiceRequest_NotFound_Returns404() {
	// ARRANGE
	suite.srDbMock.
		On("FindDocument", mock.Anything, "missing-id").
		Return(nil, db_service.ErrNotFound)

	body := `{"title":"Updated title"}`
	ctx, rec := suite.makeCtx("PUT", "/v1/service-requests/missing-id", body, gin.Params{
		{Key: "requestId", Value: "missing-id"},
	})

	// ACT
	suite.sut.UpdateServiceRequest(ctx)

	// ASSERT
	suite.Equal(http.StatusNotFound, rec.Code)
	suite.srDbMock.AssertNotCalled(suite.T(), "UpdateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *ServiceRequestsAPISuite) Test_UpdateServiceRequest_ClosedRequest_Returns409() {
	// ARRANGE
	closedSr := sampleServiceRequestDoc()
	closedSr.Status = CLOSED
	suite.srDbMock.
		On("FindDocument", mock.Anything, "sr-1").
		Return(closedSr, nil)

	body := `{"title":"Updated title"}`
	ctx, rec := suite.makeCtx("PUT", "/v1/service-requests/sr-1", body, gin.Params{
		{Key: "requestId", Value: "sr-1"},
	})

	// ACT
	suite.sut.UpdateServiceRequest(ctx)

	// ASSERT
	suite.Equal(http.StatusConflict, rec.Code)
	suite.srDbMock.AssertNotCalled(suite.T(), "UpdateDocument", mock.Anything, mock.Anything, mock.Anything)
}

// --- DeleteServiceRequest ---

func (suite *ServiceRequestsAPISuite) Test_DeleteServiceRequest_NewRequest_Returns204() {
	// ARRANGE
	suite.srDbMock.
		On("FindDocument", mock.Anything, "sr-1").
		Return(sampleServiceRequestDoc(), nil)
	suite.srDbMock.
		On("DeleteDocument", mock.Anything, "sr-1").
		Return(nil)

	ctx, _ := suite.makeCtx("DELETE", "/v1/service-requests/sr-1", "", gin.Params{
		{Key: "requestId", Value: "sr-1"},
	})

	// ACT
	suite.sut.DeleteServiceRequest(ctx)

	// ASSERT
	suite.Equal(http.StatusNoContent, ctx.Writer.Status())
	suite.srDbMock.AssertCalled(suite.T(), "DeleteDocument", mock.Anything, "sr-1")
}

func (suite *ServiceRequestsAPISuite) Test_DeleteServiceRequest_AssignedRequest_Returns409() {
	// ARRANGE
	assignedSr := sampleServiceRequestDoc()
	assignedSr.Status = ASSIGNED
	suite.srDbMock.
		On("FindDocument", mock.Anything, "sr-1").
		Return(assignedSr, nil)

	ctx, rec := suite.makeCtx("DELETE", "/v1/service-requests/sr-1", "", gin.Params{
		{Key: "requestId", Value: "sr-1"},
	})

	// ACT
	suite.sut.DeleteServiceRequest(ctx)

	// ASSERT
	suite.Equal(http.StatusConflict, rec.Code)
	suite.srDbMock.AssertNotCalled(suite.T(), "DeleteDocument", mock.Anything, mock.Anything)
}

func (suite *ServiceRequestsAPISuite) Test_DeleteServiceRequest_NotFound_Returns404() {
	// ARRANGE
	suite.srDbMock.
		On("FindDocument", mock.Anything, "missing-id").
		Return(nil, db_service.ErrNotFound)

	ctx, rec := suite.makeCtx("DELETE", "/v1/service-requests/missing-id", "", gin.Params{
		{Key: "requestId", Value: "missing-id"},
	})

	// ACT
	suite.sut.DeleteServiceRequest(ctx)

	// ASSERT
	suite.Equal(http.StatusNotFound, rec.Code)
}

