package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thebowwman/delitrack/internals/auth"
	"github.com/thebowwman/delitrack/internals/domain"
	"github.com/thebowwman/delitrack/internals/hub"
	"github.com/thebowwman/delitrack/internals/store"
)

type createDeliveryReq struct {
	OrderID         string   `json:"order_id"`
	CustomerLat     float64  `json:"customer_lat"`
	CustomerLng     float64  `json:"customer_lng"`
	CustomerAddress string   `json:"customer_address,omitempty"`
	Notes           string   `json:"notes,omitempty"`
	TTLMinutes      int      `json:"ttl_minutes,omitempty"`
	Tags            []string `json:"tags,omitempty"`
}

type createDeliveryResp struct {
	DeliveryID    string `json:"delivery_id"`
	DriverToken   string `json:"driver_token"`
	CustomerToken string `json:"customer_token"`
	WSURL         string `json:"ws_url"`
}

func handleCreateDelivery(c *gin.Context) {
	var req createDeliveryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, "bad json")
		return
	}
	if req.OrderID == "" {
		c.String(http.StatusBadRequest, "order_id required")
		return
	}

	id := hub.RandID(12)
	h := hub.GetOrCreateHub(id)
	custLoc := domain.Location{Lat: req.CustomerLat, Lng: req.CustomerLng, At: time.Now()}
	h.SetCustLoc(custLoc)

	// persist a lightweight delivery record (in-memory store for now)
	d := &domain.Delivery{
		ID:        id,
		OrderID:   req.OrderID,
		Status:    domain.StatusCreated,
		Customer:  custLoc,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.Deliveries.Create(d)

	ttl := 4 * time.Hour
	if req.TTLMinutes > 0 {
		ttl = time.Duration(req.TTLMinutes) * time.Minute
	}
	dTok, _ := auth.MakeToken(id, auth.RoleDriver, ttl)
	cTok, _ := auth.MakeToken(id, auth.RoleCustomer, ttl)

	c.JSON(http.StatusOK, createDeliveryResp{
		DeliveryID:    id,
		DriverToken:   dTok,
		CustomerToken: cTok,
		WSURL:         "ws://" + c.Request.Host + "/v1/ws/" + id,
	})
}

type wsMsg struct {
	Type     string  `json:"type"`
	Lat      float64 `json:"lat,omitempty"`
	Lng      float64 `json:"lng,omitempty"`
	Speed    float64 `json:"speed,omitempty"`
	Heading  float64 `json:"heading,omitempty"`
	Accuracy float64 `json:"accuracy,omitempty"`
	AtMs     int64   `json:"at_ms,omitempty"`
}

func tsOrNow(ms int64) time.Time {
	if ms > 0 {
		return time.UnixMilli(ms)
	}
	return time.Now()
}

// Driver posts last-known location (REST fallback)
func handlePostDriverLoc(c *gin.Context) {
	claims, err := auth.ParseTokenFromRequest(c.Request)
	if err != nil || claims.Role != auth.RoleDriver {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("deliveryID")
	if id != claims.DeliveryID {
		c.String(http.StatusForbidden, "delivery mismatch")
		return
	}
	var m wsMsg
	if err := c.ShouldBindJSON(&m); err != nil {
		c.String(http.StatusBadRequest, "bad json")
		return
	}
	loc := domain.Location{Lat: m.Lat, Lng: m.Lng, Speed: m.Speed, Heading: m.Heading, Accuracy: m.Accuracy, At: tsOrNow(m.AtMs)}
	if !loc.IsValid() {
		c.String(http.StatusBadRequest, "bad coords")
		return
	}
	h := hub.GetOrCreateHub(id)
	h.SetDriverLoc(loc)
	h.Broadcast(struct {
		Type string `json:"type"`
		domain.Location
	}{Type: "driver_loc", Location: loc}, func(c *hub.WSClient) bool { return c.Role() == auth.RoleCustomer })
	c.Status(http.StatusNoContent)
}

// Any role fetches driver's last-known location
func handleGetDriverLoc(c *gin.Context) {
	claims, err := auth.ParseTokenFromRequest(c.Request)
	if err != nil {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("deliveryID")
	if id != claims.DeliveryID {
		c.String(http.StatusForbidden, "delivery mismatch")
		return
	}
	loc := hub.GetOrCreateHub(id).GetDriverLoc()
	if loc == nil {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusOK, struct {
		Type string `json:"type"`
		*domain.Location
	}{Type: "driver_loc", Location: loc})
}

// Any role fetches customer's last-known location (typically for driver)
func handleGetCustomerLoc(c *gin.Context) {
	claims, err := auth.ParseTokenFromRequest(c.Request)
	if err != nil {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("deliveryID")
	if id != claims.DeliveryID {
		c.String(http.StatusForbidden, "delivery mismatch")
		return
	}
	loc := hub.GetOrCreateHub(id).GetCustLoc()
	if loc == nil {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusOK, struct {
		Type string `json:"type"`
		*domain.Location
	}{Type: "customer_loc", Location: loc})
}

func handleGetDelivery(c *gin.Context) {
	claims, err := auth.ParseTokenFromRequest(c.Request)
	if err != nil {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("deliveryID")
	if id != claims.DeliveryID {
		c.String(http.StatusForbidden, "delivery mismatch")
		return
	}
	if d, ok := store.Deliveries.Get(id); ok {
		c.JSON(http.StatusOK, d)
		return
	}
	c.Status(http.StatusNotFound)
}
