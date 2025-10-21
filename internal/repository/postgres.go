package repository

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq" // side-effect import: registers "postgres" driver for database/sql
	"github.com/bharathbbg/delivery-service/internal/config"
	"github.com/bharathbbg/delivery-service/internal/model"
	"github.com/google/uuid"
	"time"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(config config.DatabaseConfig) (*PostgresRepository, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresRepository{db: db}, nil
}

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

func (r *PostgresRepository) CreateDelivery(ctx context.Context, delivery *model.Delivery) (*model.Delivery, error) {
	// Generate tracking number and other necessary fields
	delivery.ID = uuid.New().String()
	delivery.TrackingNumber = fmt.Sprintf("TRK-%s", uuid.New().String()[:8])
	delivery.Status = "PENDING"
	delivery.CreatedAt = time.Now()
	delivery.UpdatedAt = time.Now()
	delivery.EstimatedDeliveryTime = time.Now().Add(72 * time.Hour) // Default: 3 days from now

	// SQL for inserting delivery
	query := `
		INSERT INTO deliveries (
			id, order_id, status, tracking_number, courier_id, 
			estimated_delivery_time, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
		RETURNING id`

	// Execute the query
	err := r.db.QueryRowContext(
		ctx,
		query,
		delivery.ID, delivery.OrderID, delivery.Status, delivery.TrackingNumber,
		delivery.CourierID, delivery.EstimatedDeliveryTime, delivery.CreatedAt, delivery.UpdatedAt,
	).Scan(&delivery.ID)

	if err != nil {
		return nil, fmt.Errorf("error creating delivery: %w", err)
	}

	// Insert shipping address in a separate table
	addressQuery := `
		INSERT INTO delivery_addresses (
			delivery_id, street, city, state, country, zip_code
		) VALUES ($1, $2, $3, $4, $5, $6)`
	
	_, err = r.db.ExecContext(
		ctx,
		addressQuery,
		delivery.ID, delivery.ShippingAddress.Street, delivery.ShippingAddress.City,
		delivery.ShippingAddress.State, delivery.ShippingAddress.Country, delivery.ShippingAddress.ZipCode,
	)

	if err != nil {
		return nil, fmt.Errorf("error creating delivery address: %w", err)
	}

	// Create initial delivery event
	eventQuery := `
		INSERT INTO delivery_events (
			id, delivery_id, status, location, description, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)`

	eventID := uuid.New().String()
	description := "Delivery created and pending processing"
	
	_, err = r.db.ExecContext(
		ctx,
		eventQuery,
		eventID, delivery.ID, delivery.Status, 
		"Warehouse", description, time.Now(),
	)

	if err != nil {
		return nil, fmt.Errorf("error creating delivery event: %w", err)
	}

	return delivery, nil
}

func (r *PostgresRepository) GetDelivery(ctx context.Context, id string) (*model.Delivery, error) {
	query := `
		SELECT 
			d.id, d.order_id, d.status, d.tracking_number, d.courier_id,
			d.estimated_delivery_time, d.actual_delivery_time, d.created_at, d.updated_at,
			a.street, a.city, a.state, a.country, a.zip_code
		FROM 
			deliveries d
		JOIN 
			delivery_addresses a ON d.id = a.delivery_id
		WHERE 
			d.id = $1`

	var delivery model.Delivery
	var street, city, state, country, zipCode sql.NullString
	var actualDeliveryTime sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&delivery.ID, &delivery.OrderID, &delivery.Status, &delivery.TrackingNumber, &delivery.CourierID,
		&delivery.EstimatedDeliveryTime, &actualDeliveryTime, &delivery.CreatedAt, &delivery.UpdatedAt,
		&street, &city, &state, &country, &zipCode,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No delivery found
		}
		return nil, err
	}

	// Handle nullable fields
	if actualDeliveryTime.Valid {
		actualTime := actualDeliveryTime.Time
		delivery.ActualDeliveryTime = &actualTime
	}

	// Set address fields
	delivery.ShippingAddress = model.Address{
		Street:  street.String,
		City:    city.String,
		State:   state.String,
		Country: country.String,
		ZipCode: zipCode.String,
	}

	return &delivery, nil
}

func (r *PostgresRepository) UpdateDelivery(ctx context.Context, req *model.UpdateDeliveryRequest) (*model.Delivery, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Update delivery status
	query := `
		UPDATE deliveries 
		SET status = $2, updated_at = $3 
		WHERE id = $1`

	now := time.Now()
	
	_, err = tx.ExecContext(ctx, query, req.ID, req.Status, now)
	if err != nil {
		return nil, err
	}

	// Create new delivery event
	eventID := uuid.New().String()
	eventQuery := `
		INSERT INTO delivery_events (
			id, delivery_id, status, location, description, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)`
	
	_, err = tx.ExecContext(
		ctx,
		eventQuery,
		eventID, req.ID, req.Status, req.Location, req.Description, now,
	)
	if err != nil {
		return nil, err
	}

	// If delivery is completed, update actual delivery time
	if req.Status == "DELIVERED" {
		completedQuery := `
			UPDATE deliveries 
			SET actual_delivery_time = $2 
			WHERE id = $1`
		
		_, err = tx.ExecContext(ctx, completedQuery, req.ID, now)
		if err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Get the updated delivery
	return r.GetDelivery(ctx, req.ID)
}

func (r *PostgresRepository) ListDeliveries(ctx context.Context, orderID string, page, pageSize int) ([]*model.Delivery, int, error) {
	// Count total matching deliveries
	countQuery := `SELECT COUNT(*) FROM deliveries WHERE ($1 = '' OR order_id = $1)`
	
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, orderID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// If total is 0, return empty slice
	if total == 0 {
		return []*model.Delivery{}, 0, nil
	}

	// Calculate offset
	offset := (page - 1) * pageSize
	
	// Query for paginated results
	query := `
		SELECT 
			d.id, d.order_id, d.status, d.tracking_number, d.courier_id,
			d.estimated_delivery_time, d.actual_delivery_time, d.created_at, d.updated_at,
			a.street, a.city, a.state, a.country, a.zip_code
		FROM 
			deliveries d
		JOIN 
			delivery_addresses a ON d.id = a.delivery_id
		WHERE 
			($1 = '' OR d.order_id = $1)
		ORDER BY 
			d.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, orderID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var deliveries []*model.Delivery

	for rows.Next() {
		var delivery model.Delivery
		var street, city, state, country, zipCode sql.NullString
		var actualDeliveryTime sql.NullTime

		err := rows.Scan(
			&delivery.ID, &delivery.OrderID, &delivery.Status, &delivery.TrackingNumber, &delivery.CourierID,
			&delivery.EstimatedDeliveryTime, &actualDeliveryTime, &delivery.CreatedAt, &delivery.UpdatedAt,
			&street, &city, &state, &country, &zipCode,
		)

		if err != nil {
			return nil, 0, err
		}

		// Handle nullable fields
		if actualDeliveryTime.Valid {
			actualTime := actualDeliveryTime.Time
			delivery.ActualDeliveryTime = &actualTime
		}

		// Set address fields
		delivery.ShippingAddress = model.Address{
			Street:  street.String,
			City:    city.String,
			State:   state.String,
			Country: country.String,
			ZipCode: zipCode.String,
		}

		deliveries = append(deliveries, &delivery)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return deliveries, total, nil
}

func (r *PostgresRepository) TrackDelivery(ctx context.Context, trackingNumber string) (*model.Delivery, []*model.DeliveryEvent, error) {
	// First get the delivery
	query := `
		SELECT 
			d.id, d.order_id, d.status, d.tracking_number, d.courier_id,
			d.estimated_delivery_time, d.actual_delivery_time, d.created_at, d.updated_at,
			a.street, a.city, a.state, a.country, a.zip_code
		FROM 
			deliveries d
		JOIN 
			delivery_addresses a ON d.id = a.delivery_id
		WHERE 
			d.tracking_number = $1`

	var delivery model.Delivery
	var street, city, state, country, zipCode sql.NullString
	var actualDeliveryTime sql.NullTime

	err := r.db.QueryRowContext(ctx, query, trackingNumber).Scan(
		&delivery.ID, &delivery.OrderID, &delivery.Status, &delivery.TrackingNumber, &delivery.CourierID,
		&delivery.EstimatedDeliveryTime, &actualDeliveryTime, &delivery.CreatedAt, &delivery.UpdatedAt,
		&street, &city, &state, &country, &zipCode,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil // No delivery found
		}
		return nil, nil, err
	}

	// Handle nullable fields
	if actualDeliveryTime.Valid {
		actualTime := actualDeliveryTime.Time
		delivery.ActualDeliveryTime = &actualTime
	}

	// Set address fields
	delivery.ShippingAddress = model.Address{
		Street:  street.String,
		City:    city.String,
		State:   state.String,
		Country: country.String,
		ZipCode: zipCode.String,
	}

	// Now get the delivery events
	eventsQuery := `
		SELECT 
			id, delivery_id, status, location, description, timestamp
		FROM 
			delivery_events
		WHERE 
			delivery_id = $1
		ORDER BY 
			timestamp ASC`

	rows, err := r.db.QueryContext(ctx, eventsQuery, delivery.ID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var events []*model.DeliveryEvent

	for rows.Next() {
		var event model.DeliveryEvent
		err := rows.Scan(
			&event.ID, &event.DeliveryID, &event.Status, 
			&event.Location, &event.Description, &event.Timestamp,
		)
		if err != nil {
			return nil, nil, err
		}
		events = append(events, &event)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	return &delivery, events, nil
}