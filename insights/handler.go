package insights

import (
    "fmt"
    "net/http"
    "github.com/gin-gonic/gin"
)

type Handler struct {
    extractor *Extractor
}

func NewHandler(extractor *Extractor) *Handler {
    return &Handler{extractor: extractor}
}

// GetDiscussionAnalysis returns analysis of all discussions
func (h *Handler) GetDiscussionAnalysis(c *gin.Context) {
    fmt.Printf("\n=== GetDiscussionAnalysis Start ===\n")
    
    chainID := c.Param("chainId")
    fmt.Printf("Request params: chainID=%s\n", chainID)
    
    if h.extractor == nil {
        fmt.Printf("Error: extractor is nil\n")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Extractor not initialized"})
        return
    }
    fmt.Printf("Extractor initialized successfully\n")

    analysis, err := h.extractor.AnalyzeDiscussions(chainID)
    if err != nil {
        fmt.Printf("Error analyzing discussions: %v\n", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if analysis == nil {
        fmt.Printf("Error: analysis is nil\n")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "No analysis generated"})
        return
    }

    fmt.Printf("Analysis generated successfully. Length: %d\n", len(analysis.Analysis))
    fmt.Printf("First 100 chars of analysis: %s...\n", analysis.Analysis[:min(100, len(analysis.Analysis))])
    fmt.Printf("=== GetDiscussionAnalysis End ===\n\n")
    c.JSON(http.StatusOK, analysis)
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
} 