package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

// MockOrderRepository is a mock for the OrderRepository
type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) Create(ctx context.Context, order models.Order, items []models.OrderItemRequest) (*models.Order, error) {
	args := m.Called(ctx, order, items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Order), args.Error(1)
}

func (m *MockOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	args := m.Called(ctx, id)
	// ... return logic for testing GetOrder ...
	return args.Get(0).(*models.Order), args.Error(1)
}

func (m *MockOrderRepository) List(ctx context.Context, status *models.OrderStatus) ([]models.Order, error) {
	args := m.Called(ctx, status)
	// ... return logic for testing ListOrders ...
	return args.Get(0).([]models.Order), args.Error(1)
}

// Add empty mock implementations for methods not used in this specific test file
func (m *MockOrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus) error {
	return nil
}
func (m *MockOrderRepository) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status models.OrderItemStatus) error {
	return nil
}

func (m *MockOrderRepository) VoidItem(ctx context.Context, itemID uuid.UUID, reason string) error {
	return nil
}
func (m *MockOrderRepository) GetStationItems(ctx context.Context, stationID uuid.UUID) ([]models.OrderItem, error) {
	args := m.Called(ctx, stationID)
	return args.Get(0).([]models.OrderItem), args.Error(1)
}
func (m *MockOrderRepository) GetOrderHistory(ctx context.Context, startDate, endDate time.Time) ([]models.Order, error) {
	args := m.Called(ctx, startDate, endDate)
	return args.Get(0).([]models.Order), args.Error(1)
}

// MockEventPublisher is a mock for the EventPublisher
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) PublishOrderCreated(context context.Context, orderID string) error {
	args := m.Called(orderID)
	return args.Error(0)
}

// --- Tests ---

func TestOrderService_CreateOrder(t *testing.T) {
	// Setup
	mockRepo := new(MockOrderRepository)
	mockPublisher := new(MockEventPublisher)

	orderService := NewOrderService(mockRepo, mockPublisher)

	userID := uuid.New()
	orderRequest := models.OrderRequest{
		Items: []models.OrderItemRequest{
			{MenuItemID: uuid.New(), Quantity: 2},
		},
	}

	createdOrder := &models.Order{
		ID:          uuid.New(),
		UserID:      userID,
		OrderNumber: "20251002-abcd",
		Status:      models.OrderStatusNew,
		OrderedAt:   time.Now(),
		Items:       []models.OrderItem{}, // For simplicity in this test
	}

	t.Run("Success Case", func(t *testing.T) {
		// Expect the repository's Create method to be called
		mockRepo.On("Create", mock.Anything, mock.Anything, orderRequest.Items).Return(createdOrder, nil).Once()

		// Expect the publisher's PublishOrderCreated method to be called with the new order's ID
		mockPublisher.On("PublishOrderCreated", createdOrder.ID.String()).Return(nil).Once()

		// Execute
		result, err := orderService.CreateOrder(context.Background(), orderRequest, userID)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, createdOrder.ID, result.ID)
		mockRepo.AssertExpectations(t)
		mockPublisher.AssertExpectations(t)
	})

	t.Run("Failure on Database Create", func(t *testing.T) {
		dbError := errors.New("database connection failed")

		// Expect the repository's Create method to be called and return an error
		mockRepo.On("Create", mock.Anything, mock.Anything, orderRequest.Items).Return(nil, dbError).Once()

		// The publisher should NOT be called if the database create fails
		mockPublisher.AssertNotCalled(t, "PublishOrderCreated", mock.Anything)

		// Execute
		result, err := orderService.CreateOrder(context.Background(), orderRequest, userID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to create order")
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success on DB Create but Failure on Publish", func(t *testing.T) {
		publishError := errors.New("NATS is down")

		// Expect the repository's Create method to succeed
		mockRepo.On("Create", mock.Anything, mock.Anything, orderRequest.Items).Return(createdOrder, nil).Once()

		// Expect the publisher's PublishOrderCreated method to be called and return an error
		mockPublisher.On("PublishOrderCreated", createdOrder.ID.String()).Return(publishError).Once()

		// Execute
		// NOTE: The service should still return the created order and a nil error,
		// because the core operation (saving the order) succeeded. The publishing failure
		// is logged as a critical error internally but doesn't fail the API call.
		result, err := orderService.CreateOrder(context.Background(), orderRequest, userID)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, createdOrder.ID, result.ID) // Order creation itself was successful
		mockRepo.AssertExpectations(t)
		mockPublisher.AssertExpectations(t)
	})
}
