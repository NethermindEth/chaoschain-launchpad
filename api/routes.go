package api

import (
	"github.com/NethermindEth/chaoschain-launchpad/api/handlers"
	"github.com/gin-gonic/gin"
	"github.com/NethermindEth/chaoschain-launchpad/insights"
)

// chainIDMiddleware injects chainID into request context
func chainIDMiddleware(chainID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First try from header, fallback to default chainID
		reqChainID := c.GetHeader("X-Chain-ID")
		if reqChainID == "" {
			reqChainID = chainID // Use default if not specified
		}
		c.Set("chainID", reqChainID)
		c.Next()
	}
}

// SetupRoutes initializes all API endpoints
func SetupRoutes(router *gin.Engine, chainID string) {
	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:4000")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, Authorization, X-Chain-Id")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	api := router.Group("/api")
	api.Use(chainIDMiddleware(chainID))
	{
		api.POST("/chains", handlers.CreateChain)
		api.GET("/chains", handlers.ListChains)
		api.POST("/register", handlers.RegisterAgent)
		api.GET("/blocks/:height", handlers.GetBlock)
		api.GET("/chain/status", handlers.GetNetworkStatus)
		api.POST("/transactions", handlers.SubmitTransaction)
		api.GET("/validators", handlers.GetValidators)
		api.GET("/social/:agentID", handlers.GetSocialStatus)
		api.POST("/validators/:agentID/influences", handlers.AddInfluence)
		api.POST("/validators/:agentID/relationships", handlers.UpdateRelationship)
		api.POST("/block/propose", handlers.ProposeBlock)
		api.GET("/forum/threads", handlers.GetAllThreads)
		blockGroup := api.Group("/blocks")
		{
			blockGroup.GET("/discussions/:blockHash", handlers.GetBlockDiscussions)
			blockGroup.GET("/discussions/height/:height", handlers.GetBlockDiscussionsByHeight)
			blockGroup.GET("/discussions", handlers.ListBlockDiscussions)
		}

		// Add insights routes
		insightsExtractor := insights.NewExtractor("")
		insightsHandler := insights.NewHandler(insightsExtractor)
		api.GET("/insights/:chainId/analysis", insightsHandler.GetDiscussionAnalysis)
	}

	// WebSocket endpoint
	router.GET("/ws", handlers.HandleWebSocket)
}
