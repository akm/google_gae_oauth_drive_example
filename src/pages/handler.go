package pages

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
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
		log.Infof(ctx, "conf URL: %q\n", req.URL.Scheme+"://"+req.URL.Host)
		log.Infof(ctx, "conf: %v\n", conf)

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
		calendarId := c.QueryParam("calendar_id")
		log.Infof(ctx, "Get events of %s from %s ti %s\n", calendarId, st, et)

		list, err := s.Events.List(calendarId).TimeMin(st).TimeMax(et).Do()
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
	log.Infof(ctx, "conf URL: %q\n", req.URL.Scheme+"://"+req.URL.Host)
	log.Infof(ctx, "conf: %v\n", conf)
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

		log.Infof(ctx, "token: %v\n", token)
		log.Infof(ctx, "token.RefreshToken: %q\n", token.RefreshToken)

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

// https://dialogflow.com/docs/fulfillment
type FulfillmentRequest struct {
	ResponseId  string                 `json:"responseId"`  // Unique id for request.
	Session     string                 `json:"session"`     // Unique session id.
	QueryResult FulfillmentQueryResult `json:"queryResult"` // Result of the conversation query or event processing.
	// OriginalDetectIntentRequest Object `json:"originalDetectIntentRequest"` // Full request coming from an integrated platform. (Facebook Messenger, Slack, etc.)
}

type FulfillmentQueryResult struct {
	QueryText                string            `json:"queryText"`                //The original text of the query.
	Parameters               map[string]string `json:"parameters"`               // Consists of parameter_name:parameter_value pairs.
	AllRequiredParamsPresent bool              `json:"allRequiredParamsPresent"` // Set to false if required parameters are missing in query.
	// FulfillmentText           String            `json:"fulfillmentText"` // Text to be pronounced to the user or shown on the screen.
	// FulfillmentMessages       Object            `json:"fulfillmentMessages"` // Collection of rich messages to show the user.
	// OutputContexts            Object            `json:"outputContexts"` // Collection of output contexts.
	// Intent                    Object            `json:"intent"` // The intent that matched the user's query.
	IntentDetectionConfidence float64 `json:"intentDetectionConfidence"` // 0-1	Matching score for the intent.
	// DiagnosticInfo         Object  `json:"diagnosticInfo"` // Free-form diagnostic info.
	LanguageCode string `json:"languageCode"` // The language that was triggered during intent matching.
}

type UserName struct {
	Email string
}

func (h *Handler) Fulfillments(c echo.Context) error {
	req := c.Request()
	ctx := appengine.NewContext(req)

	log.Infof(ctx, "Fulfillments called\n")

	var fReq FulfillmentRequest
	err := c.Bind(&fReq)
	if err != nil {
		log.Errorf(ctx, "Error bind cause of %v\n", err)
		return c.JSON(http.StatusOK, map[string]string{"fulfillmentText": "Bindに失敗しました"})
	}

	params := fReq.QueryResult.Parameters
	personName := params["person-name"]
	if personName == "" {
		return c.JSON(http.StatusOK, map[string]string{"fulfillmentText": fmt.Sprintf("person-nameが見つかりませんでした in %v", params)})
	}

	var userName UserName
	nameKey := datastore.NewKey(ctx, "UserNames", personName, 0, nil)
	err = datastore.Get(ctx, nameKey, &userName)
	if err != nil {
		log.Errorf(ctx, "Error failed to get name for %v cause of %v\n", personName, err)
		return c.JSON(http.StatusOK, map[string]string{"fulfillmentText": fmt.Sprintf("person-name %s のメールアドレスが見つかりませんでした", personName)})
	}
	email := userName.Email

	var token oauth2.Token
	key := datastore.NewKey(ctx, "UserTokens", email, 0, nil)
	err = datastore.Get(ctx, key, &token)

	if err != nil {
		log.Errorf(ctx, "Error bind cause of %v\n", err)
		return c.JSON(http.StatusOK, map[string]string{"fulfillmentText": fmt.Sprintf("person-name %s のトークンが見つかりませんでした", personName)})
	}

	conf := h.OAuth2Config(ctx, req.URL.Scheme+"://"+req.URL.Host)
	log.Infof(ctx, "conf URL: %q\n", req.URL.Scheme+"://"+req.URL.Host)
	log.Infof(ctx, "conf: %v\n", conf)
	client := conf.Client(ctx, &token)

	// // !!! IMPORTANT !!!
	// // You must add your App Engine default service account to your calendar

	// // https://github.com/google/google-api-go-client#application-default-credentials-example
	// client, err := google.DefaultClient(ctx, calendar.CalendarScope)
	// if err != nil {
	// 	log.Errorf(ctx, "Failed to create DefaultClient\n")
	// 	return err
	// }

	s, err := calendar.New(client)
	if err != nil {
		log.Errorf(ctx, "ERROR failed to get calendar.Service because of %v\n", err)
		return c.JSON(http.StatusOK, map[string]string{"fulfillmentText": "カレンダーサービスの取得に失敗しました"})
	}
	dt, err := time.Parse(time.RFC3339, params["date"])
	if err != nil {
		dt = time.Now()
	}
	d := dt.Format("2006-01-02")
	st := d + "T00:00:00+09:00"
	et := d + "T23:59:59+09:00"
	log.Infof(ctx, "Getting events of %s from %s ti %s\n", email, st, et)

	list, err := s.Events.List(email).TimeMin(st).TimeMax(et).Do()
	if err != nil {
		log.Errorf(ctx, "ERROR failed to get List by calendar.Service.Events.List because of %v\n", err)
		return c.JSON(http.StatusOK, map[string]string{"fulfillmentText": "カレンダーからイベントの取得に失敗しました"})
	}

	lines := []string{}
	for _, event := range list.Items {
		var st, et string
		if event.Start != nil {
			st = FromRFC3339ToBiz(event.Start.DateTime)
		}
		if event.End != nil {
			et = FromRFC3339ToBiz(event.End.DateTime)
		}
		lines = append(lines, fmt.Sprintf("%s から %s %s", st, et, event.Summary))
	}

	return c.JSON(http.StatusOK, map[string]string{"fulfillmentText": strings.Join(lines, "\n")})
}

// https://golang.org/pkg/time/#pkg-constants
const BizTimeFormat = "15:04"

func FromRFC3339ToBiz(src string) string {
	t, err := time.Parse(time.RFC3339, src)
	if err != nil {
		return src
	}
	return t.Format(BizTimeFormat)
}
