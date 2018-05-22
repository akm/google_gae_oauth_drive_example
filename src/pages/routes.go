package pages

import (
	"html/template"
	"os"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/middleware"
)

var e *echo.Echo

func SetupRoutes(echo *echo.Echo, dir string) {
	e = echo
	e.Use(session.Middleware(sessions.NewCookieStore([]byte(os.Getenv("SECRET_KEY_BASE")))))
	e.Use(middleware.BodyDump(bodyDumpHandler))

	views := &HandlerViews{
		templates: template.Must(template.ParseGlob(dir + "/*.html")),
	}

	h := &Handler{Views: views}

	e.GET("/", h.Index)
	e.GET("/oauth2callback", h.Callback)
	e.POST("/fulfillments", h.Fulfillments)
}

func bodyDumpHandler(c echo.Context, reqBody, resBody []byte) {
	req := c.Request()
	ctx := appengine.NewContext(req)

	log.Debugf(ctx, "Request Body\n%s\n", string(reqBody))
}
