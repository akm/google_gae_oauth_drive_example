package pages

import (
	"html/template"

	"github.com/labstack/echo"
)

var e *echo.Echo

func SetupRoutes(echo *echo.Echo, dir string) {
	e = echo

	views := &HandlerViews{
		templates: template.Must(template.ParseGlob(dir + "/*.html")),
	}

	h := &Handler{Views: views}

	e.GET("/", h.Index)
}
