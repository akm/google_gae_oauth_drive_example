package pages

import (
	"html/template"
	"os"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
)

var e *echo.Echo

func SetupRoutes(echo *echo.Echo, dir string) {
	e = echo
	e.Use(session.Middleware(sessions.NewCookieStore([]byte(os.Getenv("SECRET_KEY_BASE")))))

	views := &HandlerViews{
		templates: template.Must(template.ParseGlob(dir + "/*.html")),
	}

	h := &Handler{Views: views}

	e.GET("/", h.Index)
	e.GET("/oauth2callback", h.Callback)
}
