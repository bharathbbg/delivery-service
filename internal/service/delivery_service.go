package service

import (
	"context"
	"errors"

	"github.com/bharathbbg/delivery-service/internal/model"
	"github.com/bharathbbg/delivery-service/internal/repository"
)

type DeliveryService struct {
	repo  *repository.PostgresRepository
	cache *repository.RedisCache
}

func NewDeliveryService(repo *repository.PostgresRepository, cache *repository.RedisCache) *DeliveryService {
	return &DeliveryService{
		repo:  repo,
		cache: cache,
	}
}

func (s *DeliveryService) CreateDelivery(ctx context.Context, req *model.CreateDeliveryRequest) (*model.Delivery, error) {
	// Validate request
	if req.OrderID == "" {
		return nil, errors.New("order_id is required")
	}

	// Create delivery object
	delivery := &model.Delivery{
		OrderID:         req.OrderID,
		ShippingAddress: req.ShippingAddress,
		Status:          "PENDING",
	}

	// Save to database
	savedDelivery, err := s.repo.CreateDelivery(ctx, delivery)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if err := s.cache.CacheDelivery(ctx, savedDelivery); err != nil {
		// Just log error, don't fail the request
		// log.Printf("Failed to cache delivery: %v", err)
	}

	// Also cache by tracking number for easy lookup
	if err := s.cache.CacheDeliveryByTracking(ctx, savedDelivery); err != nil {
		// log.Printf("Failed to cache delivery by tracking: %v", err)
	}

	return savedDelivery, nil
}

func (s *DeliveryService) GetDelivery(ctx context.Context, id string) (*model.Delivery, error) {
	// Try to get from cache first
	cachedDelivery, err := s.cache.GetCachedDelivery(ctx, id)
	if err == nil && cachedDelivery != nil {
		return cachedDelivery, nil
	}

	// If not in cache, get from database
	delivery, err := s.repo.GetDelivery(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result for future requests
	if delivery != nil {
		if err := s.cache.CacheDelivery(ctx, delivery); err != nil {
			// log.Printf("Failed to cache delivery: %v", err)
		}
	}

	return delivery, nil
}

func (s *DeliveryService) UpdateDelivery(ctx context.Context, req *model.UpdateDeliveryRequest) (*model.Delivery, error) {
	// Validate request
	if req.ID == "" {
		return nil, errors.New("delivery_id is required")
	}
	if req.Status == "" {
		return nil, errors.New("status is required")
	}

	// Update in database
	updatedDelivery, err := s.repo.UpdateDelivery(ctx, req)
	if err != nil {
		return nil, err
	}

	if updatedDelivery == nil {
		return nil, errors.New("delivery not found")
	}

	// Invalidate and update cache
	if err := s.cache.CacheDelivery(ctx, updatedDelivery); err != nil {
		// log.Printf("Failed to update delivery cache: %v", err)
	}
	if err := s.cache.CacheDeliveryByTracking(ctx, updatedDelivery); err != nil {
		// log.Printf("Failed to update delivery tracking cache: %v", err)
	}

	return updatedDelivery, nil
}

func (s *DeliveryService) ListDeliveries(ctx context.Context, orderID string, page, pageSize int) ([]*model.Delivery, int, error) {
	// Ensure valid pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// Get from database
	return s.repo.ListDeliveries(ctx, orderID, page, pageSize)
}

func (s *DeliveryService) TrackDelivery(ctx context.Context, trackingNumber string) (*model.Delivery, []*model.DeliveryEvent, error) {
	// Try to get from cache first
	cachedDelivery, err := s.cache.GetCachedDeliveryByTracking(ctx, trackingNumber)
	if err == nil && cachedDelivery != nil {
		// If delivery is in cache, try to get events from cache
		events, err := s.cache.GetCachedDeliveryEvents(ctx, cachedDelivery.ID)
		if err == nil && events != nil {
			return cachedDelivery, events, nil
		}
	}

	// If not in cache or events not in cache, get from database
	delivery, events, err := s.repo.TrackDelivery(ctx, trackingNumber)
	if err != nil {
		return nil, nil, err
	}

	if delivery == nil {
		return nil, nil, nil
	}

	// Cache the results
	if err := s.cache.CacheDelivery(ctx, delivery); err != nil {
		// log.Printf("Failed to cache delivery: %v", err)
	}
	if err := s.cache.CacheDeliveryByTracking(ctx, delivery); err != nil {
		// log.Printf("Failed to cache delivery by tracking: %v", err)
	}
	if err := s.cache.CacheDeliveryEvents(ctx, delivery.ID, events); err != nil {
		// log.Printf("Failed to cache delivery events: %v", err)
	}

	return delivery, events, nil
}
