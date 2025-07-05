package http

import "github.com/gin-gonic/gin"

func NewRouter(handler *HackingHandler) *gin.Engine {
	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})
	api := router.Group("/v1")
	{
		api.GET("/infos", handler.GetTimeline)
		api.GET("/tags", handler.ListTags)
		api.POST("/simulate-scraping", handler.SimulateScraping)
	}
	return router
}