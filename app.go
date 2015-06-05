package mylibraweight

import (
	"appengine"
	"appengine/urlfetch"
	"fmt"
	// "golang.org/x/oauth2"
	// "golang.org/x/oauth2/google"
	"code.google.com/p/goauth2/oauth"
	// "google.golang.org/api/drive/v2"
	"html/template"
	"net/http"
)

// Cache all of the HTML files in the templates directory so that we only have to hit disk once.
var cached_templates = template.Must(template.ParseGlob("templates/*.html"))

var conf = &oauth.Config{
	ClientId:     "285312328170-a54o8ukf7lmlan610vfh1cr4iq4boemp.apps.googleusercontent.com", // from https://console.developers.google.com/project/<your-project-id>/apiui/credential
	ClientSecret: "IAFj6KxoAyYRKGYHiPK4I88Z",
	AuthURL:      "https://accounts.google.com/o/oauth2/auth",
	RedirectURL:  "http://mylibraweight.appspot.com/oauth2callback",
	TokenURL:     "https://accounts.google.com/o/oauth2/token",
	Scope:        "https://www.googleapis.com/auth/drive",
}

// This is the URL that Google has defined so that an authenticated application may obtain the user's info in json format.
const profileInfoURL = "https://www.googleapis.com/oauth2/v1/userinfo?alt=json"

// init is called before the application starts.
func init() {
	// Register a handler for /hello URLs.
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/authorize", handleAuthorize)
	http.HandleFunc("/oauth2callback/", handleOAuth2Callback)
}

//
func handleRoot(w http.ResponseWriter, r *http.Request) {

	err := cached_templates.ExecuteTemplate(w, "notAuthenticated.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}

}

//
func handleAuthorize(w http.ResponseWriter, r *http.Request) {
	// Get the Google URL which shows the Authentication page to the user.
	url := conf.AuthCodeURL("")

	// Redirect user to that page.
	http.Redirect(w, r, url, http.StatusFound)
}

//
func handleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	// Initialize an appengine context.
	c := appengine.NewContext(r)

	// Retrieve the code from the response.
	code := r.FormValue("code")

	// Configure OAuth's http.Client to use the appengine/urlfetch transport that all Google App Engine applications have to use for outbound requests.
	t := &oauth.Transport{Config: conf, Transport: &urlfetch.Transport{Context: c}}

	// Exchange the received code for a token.
	token, err := t.Exchange(code)
	if err != nil {
		c.Errorf("%v", err)
	} else {
		// Log the token
		c.Infof("Token: %s", token.AccessToken)
	}

	// Now get user data based on the Transport which has the token.
	resp, _ := t.Client().Get(fmt.Sprint(profileInfoURL, "&access_token=", token.AccessToken))
	buf := make([]byte, 1024)
	resp.Body.Read(buf)

	// Render the user's information.
	err = cached_templates.ExecuteTemplate(w, "userInfo.html", string(buf))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
}
