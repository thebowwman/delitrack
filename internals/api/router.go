package api

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) { c.Status(200) })

	v1 := r.Group("/v1")
	{
		v1.POST("/deliveries", handleCreateDelivery)
		v1.GET("/ws/:deliveryID", handleWS)
		v1.GET("/deliveries/:deliveryID", handleGetDelivery)
		v1.POST("/deliveries/:deliveryID/driver/location", handlePostDriverLoc)
		v1.GET("/deliveries/:deliveryID/driver/location", handleGetDriverLoc)
		v1.GET("/deliveries/:deliveryID/customer/location", handleGetCustomerLoc)
	}
}
