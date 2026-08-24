package foo

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type createFooRequest struct {
	Name string `json:"name" binding:"required"`
}

// @Summary      Create foo
// @Description  Creates a Foo by calling example.Service in-process
// @Tags         foo
// @Accept       json
// @Produce      json
// @Param        request  body      createFooRequest  true  "foo payload"
// @Success      201  {object}  Foo
// @Router       /foo [post]
func (d *dependencies) HandleFoo(c *gin.Context) {
	var req createFooRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(fmt.Errorf("bind create foo request: %w", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ex, err := d.example.CreateExample(c.Request.Context(), req.Name)
	if err != nil {
		_ = c.Error(fmt.Errorf("create foo %q: %w", req.Name, err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, &Foo{Example: ex})
}
