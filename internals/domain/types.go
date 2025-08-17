package domain

import (
	"math"
	"time"
)

type Location struct {
	Lat      float64   `json:"lat"`
	Lng      float64   `json:"lng"`
	Speed    float64   `json:"speed"`
	Heading  float64   `json:"heading,omitempty"`
	Accuracy float64   `json:"accuracy,omitempty"`
	At       time.Time `json:"at"`
}

func (l Location) IsValid() bool {

	return !math.IsNaN(l.Lat) && !math.IsNaN(l.Lng) && l.Lat <= 90 && l.Lat >= -90 && l.Lng <= 180 && l.Lng >= -180

}

type Delivery struct {
	ID             string    `json:"id"`
	OrderID        string    `json:"order_id"`
	Status         string    `json:"status"`
	AssignedDriver string    `json:"assigned_driver,omitempty"`
	Customer       Location  `json:"customer"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

const (
	StatusCreated   = "created"
	StatusPickedUp  = "picked_up"
	StatusDelivered = "delivered"
)
