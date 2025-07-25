package http

import (
	"fmt"
	"github.com/itout-datetoya/hack-info-timeline/usecases"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type HackingHandler struct {
	hackingUsecase *usecases.HackingUsecase
}

func NewHackingHandler(hackingUsecase *usecases.HackingUsecase) *HackingHandler {
	return &HackingHandler{hackingUsecase: hackingUsecase}
}

func (h *HackingHandler) GetLatestTimeline(c *gin.Context) {
	tagsQuery := c.Query("tags")
	infoNumberQuery := c.Query("infoNumber")

	var tags []string
	if tagsQuery != "" {
		tags = strings.Split(tagsQuery, ",")
	}
	infoNumber, err := strconv.Atoi(infoNumberQuery)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid infoNumber format"})
		return
	}

	infos, err := h.hackingUsecase.GetLatestTimeline(c.Request.Context(), tags, infoNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		log.Printf("Failed to get latest hacking timeline: %v", err)
		return
	}
	c.JSON(http.StatusOK, infos)
}

func (h *HackingHandler) GetPrevTimeline(c *gin.Context) {
	tagsQuery := c.Query("tags")
	prevInfoIDQuery := c.Query("prevInfoID")
	infoNumberQuery := c.Query("infoNumber")

	var tags []string
	if tagsQuery != "" {
		tags = strings.Split(tagsQuery, ",")
	}

	prevInfoID, err := strconv.ParseInt(prevInfoIDQuery, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prevInfoID format"})
		return
	}

	infoNumber, err := strconv.Atoi(infoNumberQuery)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid infoNumber format"})
		return
	}

	infos, err := h.hackingUsecase.GetPrevTimeline(c.Request.Context(), tags, prevInfoID, infoNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		log.Printf("Failed to get previous hacking timeline: %v", err)
		return
	}
	c.JSON(http.StatusOK, infos)
}

func (h *HackingHandler) GetAllTags(c *gin.Context) {
	tags, err := h.hackingUsecase.GetAllTags(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		log.Printf("Failed to get Tags: %v", err)
		return
	}
	c.JSON(http.StatusOK, tags)
}

func (h *HackingHandler) ScrapeNewInfos(c *gin.Context) {
	limitQuery := c.Query("limit")
	limit, err := strconv.Atoi(limitQuery)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid infoNumber format"})
		return
	}

	processedCount, errs := h.hackingUsecase.ScrapeAndStore(c.Request.Context(), limit)

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
