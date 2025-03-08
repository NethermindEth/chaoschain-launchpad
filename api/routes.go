package api

import (
	"github.com/NethermindEth/chaoschain-launchpad/api/handlers"
	"github.com/gin-gonic/gin"
)

// SetupRoutes initializes all API endpoints
func SetupRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		api.POST("/register", handlers.RegisterAgent)
		api.GET("/blocks/:height", handlers.GetBlock)
		api.GET("/chain/status", handlers.GetNetworkStatus)
		api.POST("/transactions", handlers.SubmitTransaction)
		api.GET("/validators", handlers.GetValidators)
		api.GET("/social/:agentID", handlers.GetSocialStatus)
		api.POST("/validators/:agentID/influences", handlers.AddInfluence)
		api.POST("/validators/:agentID/relationships", handlers.UpdateRelationship)
		api.POST("/block/propose", handlers.ProposeBlock)
	}
}
