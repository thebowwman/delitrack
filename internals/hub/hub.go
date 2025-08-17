package hub

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/thebowwman/delitrack/internals/auth"
	"github.com/thebowwman/delitrack/internals/domain"
)

type DeliveryHub struct {
	ID            string
	mu            sync.RWMutex
	clients       map[*WSClient]struct{}
	lastDriverLoc *domain.Location
	lastCustLoc   *domain.Location
}

func NewHub(id string) *DeliveryHub {

	return &DeliveryHub{
		ID:      id,
		clients: make(map[*WSClient]struct{}),
	}
}

func (h *DeliveryHub) AddClient(c *WSClient) {

	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *DeliveryHub) RemoveClient(c *WSClient) {

	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *DeliveryHub) Broadcast(msg any, filter func(*WSClient) bool) {

	b, _ := json.Marshal(msg)

	h.mu.RLock()
	for c := range h.clients {

		if filter == nil || filter(c) {
			c.Send(b)
		}
	}

	h.mu.RUnlock()
}

func (h *DeliveryHub) SetDriverLoc(loc domain.Location) {
	h.mu.Lock()
	h.lastDriverLoc = &loc
	h.mu.Unlock()
}

func (h *DeliveryHub) SetCustLoc(loc domain.Location) {
	h.mu.Lock()
	h.lastCustLoc = &loc
	h.mu.Unlock()
}

func (h *DeliveryHub) GetDriverLoc() *domain.Location {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastDriverLoc
}

func (h *DeliveryHub) GetCustLoc() *domain.Location {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastCustLoc
}

// --- lightweight registry (in-memory) ---
var hubs = struct{ sync.Map }{}

func GetOrCreateHub(id string) *DeliveryHub {

	if v, ok := hubs.Load(id); ok {
		return v.(*DeliveryHub)
	}
	h := NewHub(id)
	v, _ := hubs.LoadOrStore(id, h)
	return v.(*DeliveryHub)

}

func RandID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)

}

type WSClient struct {
	conn *websocket.Conn
	role auth.Role
	hub  *DeliveryHub
	mu   sync.Mutex
}

func NewWSClient(conn *websocket.Conn, role auth.Role, h *DeliveryHub) *WSClient {

	return &WSClient{
		conn: conn,
		role: role,
		hub:  h,
	}
}
func (c *WSClient) Send(b []byte) {

	c.mu.Lock()
	defer c.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = c.conn.Write(ctx, websocket.MessageText, b)

}

func (c *WSClient) Role() auth.Role { return c.role }

func (c *WSClient) SendJSON(typ string, loc domain.Location) {

	msg := struct {
		Type string          `json:"type"`
		Loc  domain.Location `json:"-"`
		domain.Location
	}{Type: typ, Location: loc}

	b, _ := json.Marshal(msg)
	c.Send(b)
}
