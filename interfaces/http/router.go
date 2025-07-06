package http

import "github.com/gin-gonic/gin"

func NewRouter(hackingHandler HackingHandler, transferHandler TransferHandler) *gin.Engine {
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
		api.GET("/hacking/infos", hackingHandler.GetHackingTimeline)
		api.GET("/transfer/infos", transferHandler.GetTransferTimeline)
		api.GET("/hacking/tags", hackingHandler.ListHackingTags)
		api.GET("/transfer/tags", transferHandler.ListTransferTags)
		api.POST("/hacking/scrape-new-infos", hackingHandler.ScrapeNewInfos)
		api.POST("/transfer/scrape-new-infos", transferHandler.ScrapeNewInfos)
	}
	return router
}