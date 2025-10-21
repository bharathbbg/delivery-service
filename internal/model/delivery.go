package model

import (
	"time"
)

type Delivery struct {
	ID                   string         `json:"id" db:"id"`
	OrderID              string         `json:"order_id" db:"order_id"`
	ShippingAddress      Address        `json:"shipping_address"`
	CourierID            string         `json:"courier_id" db:"courier_id"`
	Status               string         `json:"status" db:"status"`
	TrackingNumber       string         `json:"tracking_number" db:"tracking_number"`
	EstimatedDeliveryTime time.Time     `json:"estimated_delivery_time" db:"estimated_delivery_time"`
	ActualDeliveryTime   *time.Time     `json:"actual_delivery_time,omitempty" db:"actual_delivery_time"`
	CreatedAt           time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at" db:"updated_at"`
}

type DeliveryEvent struct {
	ID          string    `json:"id" db:"id"`
	DeliveryID  string    `json:"delivery_id" db:"delivery_id"`
	Status      string    `json:"status" db:"status"`
	Location    string    `json:"location" db:"location"`
	Description string    `json:"description" db:"description"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
}

type Address struct {
	Street  string `json:"street" db:"street"`
	City    string `json:"city" db:"city"`
	State   string `json:"state" db:"state"`
	Country string `json:"country" db:"country"`
	ZipCode string `json:"zip_code" db:"zip_code"`
}

// Request/Response models
type CreateDeliveryRequest struct {
	OrderID         string  `json:"order_id" binding:"required"`
	ShippingAddress Address `json:"shipping_address" binding:"required"`
}

type UpdateDeliveryRequest struct {
	ID          string `json:"-"`
	Status      string `json:"status" binding:"required"`
	Location    string `json:"location"`
	Description string `json:"description"`
}