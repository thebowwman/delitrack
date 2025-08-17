package api

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/thebowwman/delitrack/internals/auth"
	"github.com/thebowwman/delitrack/internals/domain"
	"github.com/thebowwman/delitrack/internals/hub"
)

func handleWS(c *gin.Context) {
	// 1) Accept JWT from Authorization header OR from `?token=` for browser clients
	claims, err := auth.ParseTokenFromRequest(c.Request)
	if err != nil {
		if tok := c.Query("token"); tok != "" {
			claims, err = auth.ParseToken(tok)
		}
	}
	if err != nil {
		c.String(401, "unauthorized")
		return
	}

	// 2) Delivery ID (supports wildcard route /ws/*deliveryID)
	deliveryID := strings.TrimPrefix(c.Param("deliveryID"), "/")
	if deliveryID == "" || deliveryID != claims.DeliveryID {
		c.String(403, "delivery mismatch")
		return
	}

	// 3) Upgrade to WebSocket
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{InsecureSkipVerify: true}) // TODO: use OriginPatterns in prod
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "bye")
	conn.SetReadLimit(1 << 20)

	// 4) Register client in the per-delivery hub
	h := hub.GetOrCreateHub(deliveryID)
	client := hub.NewWSClient(conn, claims.Role, h)
	h.AddClient(client)
	defer h.RemoveClient(client)

	// 5) On connect, push counterpart's last known location
	if claims.Role == auth.RoleCustomer {
		if loc := h.GetDriverLoc(); loc != nil {
			client.SendJSON("driver_loc", *loc)
		}
	} else if claims.Role == auth.RoleDriver {
		if loc := h.GetCustLoc(); loc != nil {
			client.SendJSON("customer_loc", *loc)
		}
	}

	// 6) Keepalive pings
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for range t.C {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = conn.Ping(ctx)
			cancel()
		}
	}()

	// 7) Read loop: process location updates and fan-out
	for {
		mt, data, err := conn.Read(context.Background())
		if err != nil {
			break
		}
		if mt != websocket.MessageText {
			continue
		}
		var m struct {
			Type     string  `json:"type"`
			Lat      float64 `json:"lat,omitempty"`
			Lng      float64 `json:"lng,omitempty"`
			Speed    float64 `json:"speed,omitempty"`
			Heading  float64 `json:"heading,omitempty"`
			Accuracy float64 `json:"accuracy,omitempty"`
			AtMs     int64   `json:"at_ms,omitempty"`
		}
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		switch m.Type {
		case "driver_loc":
			if claims.Role != auth.RoleDriver {
				continue
			}
			loc := domain.Location{Lat: m.Lat, Lng: m.Lng, Speed: m.Speed, Heading: m.Heading, Accuracy: m.Accuracy, At: tsOrNow(m.AtMs)}
			if !loc.IsValid() {
				continue
			}
			h.SetDriverLoc(loc)
			h.Broadcast(struct {
				Type string `json:"type"`
				domain.Location
			}{Type: "driver_loc", Location: loc}, func(c *hub.WSClient) bool { return c.Role() == auth.RoleCustomer })
		case "customer_loc":
			if claims.Role != auth.RoleCustomer {
				continue
			}
			loc := domain.Location{Lat: m.Lat, Lng: m.Lng, Speed: m.Speed, Heading: m.Heading, Accuracy: m.Accuracy, At: tsOrNow(m.AtMs)}
			if !loc.IsValid() {
				continue
			}
			h.SetCustLoc(loc)
			h.Broadcast(struct {
				Type string `json:"type"`
				domain.Location
			}{Type: "customer_loc", Location: loc}, func(c *hub.WSClient) bool { return c.Role() == auth.RoleDriver })
		}
	}
}
