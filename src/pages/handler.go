package pages

import (
	// "fmt"
	"net/http"

	// "models"

	"github.com/labstack/echo"
	// "golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

type Handler struct {
	Views Views
}

type IndexData struct {
}

func (h *Handler) Index(c echo.Context) error {
	ctx := appengine.NewContext(c.Request())
	log.Infof(ctx, "Handler#Index #0\n")
	r := &IndexData{}
	log.Infof(ctx, "Handler#Index #1\n")
	res := h.Views.Render(c, http.StatusOK, "index", r)
	log.Infof(ctx, "Handler#Index #2 res: %v\n", res)
	return res
}
