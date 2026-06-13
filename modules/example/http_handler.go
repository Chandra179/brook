package example

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (d *Dependencies) Handle(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
