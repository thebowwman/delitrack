package store

import (
	"errors"
	"sync"

	"github.com/thebowwman/delitrack/internals/domain"
)

type DeliveryStore struct {
	mu sync.RWMutex
	m  map[string]*domain.Delivery
}

func NewDeliveryStore() *DeliveryStore { return &DeliveryStore{m: make(map[string]*domain.Delivery)} }

var Deliveries = NewDeliveryStore()

func (s *DeliveryStore) Create(d *domain.Delivery) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[d.ID] = d
}

func (s *DeliveryStore) Get(id string) (*domain.Delivery, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.m[id]
	return d, ok
}

func (s *DeliveryStore) Update(d *domain.Delivery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[d.ID]; !ok {
		return errors.New("not found")
	}
	s.m[d.ID] = d
	return nil
}
