package http

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"github.com/itout-datetoya/hack-info-timeline/usecases"

	"github.com/gin-gonic/gin"
)

type HackingHandler struct {
	hackingUsecase *usecases.HackingUsecase
}

func NewHackingHandler(hackingUsecase *usecases.HackingUsecase) *HackingHandler {
	 return &HackingHandler{hackingUsecase: hackingUsecase} 
	}

func (h *HackingHandler) GetHackingTimeline(c *gin.Context) {
	tagsQuery := c.Query("tags")
	var tags []string
	if tagsQuery != "" {
		tags = strings.Split(tagsQuery, ",")
	}
	infos, err := h.hackingUsecase.GetTimeline(c.Request.Context(), tags)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	c.JSON(http.StatusOK, infos)
}

func (h *HackingHandler) ListHackingTags(c *gin.Context) {
	tags, err := h.hackingUsecase.GetAllTags(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	c.JSON(http.StatusOK, tags)
}

func (h *HackingHandler) SimulateScraping(c *gin.Context) {
	processedCount, errs := h.hackingUsecase.ScrapeAndStore(c.Request.Context())

	if len(errs) > 0 {
		// エラーはサーバー側でログに記録
		for _, err := range errs {
			log.Printf("Scraping error: %v", err)
		}
		// クライアントにはエラーがあったことと件数を返す
		c.JSON(http.StatusInternalServerError, gin.H{
			"message":         fmt.Sprintf("Scraping completed with %d errors.", len(errs)),
			"processed_count": processedCount,
			"error_count":     len(errs),
		})
		return
	}

	if processedCount == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":         "No new messages to process.",
			"processed_count": 0,
			"error_count":     0,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         fmt.Sprintf("Successfully processed %d new infos.", processedCount),
		"processed_count": processedCount,
		"error_count":     0,
	})
}