package debug

import (
	"github.com/gin-gonic/gin"
)


// SetupDebugRoutes sets up the debug routes
func SetupDebugRoutes(g *gin.RouterGroup) {
    g.GET("/ping", ping)
    g.GET("/healthcheck", healthCheck)
    g.POST("/reset", resetDatabase)
}

