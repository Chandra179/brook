package example

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary      Example endpoint
// @Description  Returns a simple status response
// @Tags         example
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /example [post]
func (d *dependencies) HandleExample(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
