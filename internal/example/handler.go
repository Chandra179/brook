package example

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"brook/middleware"
)

type createExampleRequest struct {
	Name string `json:"name" binding:"required"`
}

// @Summary      Create example
// @Description  Creates and persists an example record
// @Tags         example
// @Accept       json
// @Produce      json
// @Param        request  body      createExampleRequest  true  "example payload"
// @Success      201  {object}  Example
// @Router       /example [post]
func (d *dependencies) HandleExample(c *gin.Context) {
	var req createExampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(fmt.Errorf("bind create example request: %w", err)).SetMeta(middleware.SafeErrorMessage("invalid create example request"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ex, err := d.CreateExample(c.Request.Context(), req.Name)
	if err != nil {
		if errors.Is(err, ErrReservedName) {
			_ = c.Error(err).SetMeta(middleware.SafeErrorMessage("name is reserved"))
			c.JSON(http.StatusConflict, gin.H{"error": "name is reserved"})
			return
		}

		_ = c.Error(err).SetMeta(middleware.SafeErrorMessage("create example failed"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, ex)
}
