package pages

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	// "models"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/user"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
)

// https://developers.google.com/identity/protocols/OAuth2WebServer
type Handler struct {
	Views Views
}

type IndexData struct {
	Content string
	Error   interface{}
}

func (h *Handler) Index(c echo.Context) error {
	req := c.Request()
	ctx := appengine.NewContext(req)
	sess, err := session.Get("session", c)
	if err != nil {
		return c.String(http.StatusOK, "ERROR failed to get session")
	}

	tokenData, ok := sess.Values["credentials"]
	log.Infof(ctx, "Session has credentials? => %v\n", ok)
	r := &IndexData{}
	r.Error = sess.Values["error"]
	if ok {
		tokenJson, ok := tokenData.(string)
		if !ok {
			return c.String(http.StatusOK, fmt.Sprintf("ERROR tokenData isn't a string but [%T] %v", tokenData, tokenData))
		}
		var token oauth2.Token
		err := json.Unmarshal([]byte(tokenJson), &token)
		if err != nil {
			log.Errorf(ctx, "ERROR failed to parse token JSON because of %v\n", err)
			return c.String(http.StatusOK, "ERROR failed to parse token JSON")
		}
		conf := h.OAuth2Config(ctx, req.URL.Scheme+"://"+req.URL.Host)
		client := conf.Client(ctx, &token)
		s, err := calendar.New(client)
		if err != nil {
			log.Errorf(ctx, "ERROR failed to get calendar.Service because of %v\n", err)
			return c.String(http.StatusOK, "ERROR failed to get calendar.Service")
		}
		st := c.QueryParam("st")
		et := c.QueryParam("et")
		if st == "" {
			st = time.Now().Format(time.RFC3339)
		}
		if et == "" {
			t, err := time.Parse(time.RFC3339, st)
			if err != nil {
				t = time.Now()
			}
			et = t.AddDate(0, 1, 0).Format(time.RFC3339)
		}
		list, err := s.Events.
			List(c.QueryParam("calendar_id")).TimeMin(st).TimeMax(et).Do()
		if err != nil {
			log.Errorf(ctx, "ERROR failed to get List by calendar.Service.CalendarList because of %v\n", err)
			return c.String(http.StatusOK, "ERROR failed to get List by calendar.Service.CalendarList")
		}
		log.Infof(ctx, "Success to get list: %v\n", list)
		bytes, err := json.MarshalIndent(list, "", "  ")
		if err != nil {
			log.Errorf(ctx, "ERROR failed to marshal list because of %v\n", err)
			return c.String(http.StatusOK, "ERROR failed to marshal list")
		}
		r.Content = string(bytes)
	}
	return h.Views.Render(c, http.StatusOK, "index", r)
}

func (h *Handler) Callback(c echo.Context) error {
	req := c.Request()
	code := c.QueryParam("code")
	ctx := appengine.NewContext(req)
	conf := h.OAuth2Config(ctx, req.URL.Scheme+"://"+req.URL.Host)
	if code == "" {
		url := conf.AuthCodeURL("state", oauth2.AccessTypeOnline)
		return c.Redirect(http.StatusFound, url)
	} else {
		sess, _ := session.Get("session", c)
		sess.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7,
			HttpOnly: true,
		}

		token, err := conf.Exchange(ctx, code)
		if err != nil {
			msg := "Failed conf.Exchange"
			sess.Values["error"] = msg
			sess.Save(c.Request(), c.Response())
			log.Errorf(ctx, "ERROR %s\n", msg)
			return c.Redirect(http.StatusFound, "/")
		}

		tokenJson, err := json.Marshal(token)
		if err != nil {
			msg := "Failed to marshal token to JSON"
			sess.Values["error"] = msg
			sess.Save(c.Request(), c.Response())
			log.Errorf(ctx, "ERROR %s\n", msg)
			return c.Redirect(http.StatusFound, "/")
		}
		sess.Values["credentials"] = string(tokenJson)
		log.Infof(ctx, "tokenJson: %v\n", string(tokenJson))

		u := user.Current(ctx)
		if u == nil {
			url, _ := user.LoginURL(ctx, "/")
			// fmt.Fprintf(w, `<a href="%s">Sign in or register</a>`, url)
			return c.Redirect(http.StatusFound, url)
		}

		key := datastore.NewKey(ctx, "UserTokens", u.Email, 0, nil)
		_, err = datastore.Put(ctx, key, token)
		if err != nil {
			log.Errorf(ctx, "ERROR %v\n", err)
			return c.Redirect(http.StatusFound, "/")
		}

		sess.Save(c.Request(), c.Response())
		return c.Redirect(http.StatusFound, "/")
	}
}

func (h *Handler) OAuth2Config(ctx context.Context, baseUrl string) *oauth2.Config {
	gcpClientID := os.Getenv("GCP_CLIENT_ID")
	gcpClientSecret := os.Getenv("GCP_CLIENT_SECRET")
	log.Infof(ctx, "gcpClientID: %q\n", gcpClientID)

	return &oauth2.Config{
		ClientID:     gcpClientID,
		ClientSecret: gcpClientSecret,
		Scopes: []string{
			drive.DriveScope,
			sheets.SpreadsheetsScope,
			calendar.CalendarScope,
		},
		Endpoint:    google.Endpoint,
		RedirectURL: baseUrl + "/oauth2callback",
	}
}
