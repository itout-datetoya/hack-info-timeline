package http

import "github.com/gin-gonic/gin"

func NewRouter(hackingHandler HackingHandler, transferHandler TransferHandler) *gin.Engine {
	router := gin.Default()
	api := router.Group("/v1")
	{
		api.GET("/hacking/latest-infos", hackingHandler.GetLatestTimeline)
		api.GET("/hacking/prev-infos", hackingHandler.GetPrevTimeline)
		api.GET("/hacking/tags", hackingHandler.GetAllTags)
		api.POST("/hacking/scrape-new-infos", hackingHandler.ScrapeNewInfos)
		
		api.GET("/transfer/latest-infos", transferHandler.GetLatestTimeline)
		api.GET("/transfer/prev-infos", transferHandler.GetPrevTimeline)
		api.GET("/transfer/tags", transferHandler.GetAllTags)
		api.POST("/transfer/scrape-new-infos", transferHandler.ScrapeNewInfos)
	}
	return router
}