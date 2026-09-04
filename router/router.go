package router

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"brook/middleware"
)

// New builds the Gin engine: middleware chain, health/swagger routes, and
// module handlers.
func (d *dependencies) New() *gin.Engine {
	r := gin.New()

	r.Use(
		gin.CustomRecovery(func(c *gin.Context, err any) {
			d.logger.Error("panic recovered",
				zap.String("panic", fmt.Sprintf("%v", err)),
				zap.String("stack", string(debug.Stack())),
			)
			c.AbortWithStatus(http.StatusInternalServerError)
		}),
		middleware.RequestID,
		d.requestLog,
	)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.POST("/example", d.example)

	return r
}
