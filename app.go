package mylibraweight

import (
	// "fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/plus/v1"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"html/template"
	"net/http"
	// "strings"
)

// Cache all of the HTML files in the templates directory so that we only have to hit disk once.
var cached_templates = template.Must(template.ParseGlob("templates/*.html"))

var conf = &oauth2.Config{
	ClientID:     "285312328170-a54o8ukf7lmlan610vfh1cr4iq4boemp.apps.googleusercontent.com", // from https://console.developers.google.com/project/<your-project-id>/apiui/credential
	ClientSecret: "IAFj6KxoAyYRKGYHiPK4I88Z",
	RedirectURL:  "http://mylibraweight.appspot.com/oauth2callback",
	// RedirectURL: "http://localhost:10080/oauth2callback",
	Scopes: []string{
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/userinfo.profile",
	},
	Endpoint: google.Endpoint,
}

// var conf = &oauth2.Config{
// 	ClientID:     "285312328170-7dvm2p1sa9tnfpfblopuk4eqp0r80jvl.apps.googleusercontent.com", // from https://console.developers.google.com/project/<your-project-id>/apiui/credential
// 	ClientSecret: "-qW7bzzoddgeXOIo-G-4H_4K",
// 	RedirectURL:  "http://localhost:10080/oauth2callback",
// 	// RedirectURL: "http://localhost:10080/oauth2callback",
// 	Scopes: []string{
// 		"https://www.googleapis.com/auth/drive",
// 		"profile",
// 	},
// 	Endpoint: google.Endpoint,
// }

// This is the URL that Google has defined so that an authenticated application may obtain the user's info in json format.
// const profileInfoURL = "https://www.googleapis.com/oauth2/v1/userinfo?alt=json"

// init is called before the application starts.
func init() {
	// Register a handler for /hello URLs.
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/authorize", handleAuthorize)
	http.HandleFunc("/oauth2callback", handleOAuth2Callback)
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
	// Initialize an appengine context.
	c := appengine.NewContext(r)
	log.Infof(c, "In handleAuthorize")

	// Get the Google URL which shows the Authentication page to the user.
	// url := conf.AuthCodeURL("state")
	url := conf.AuthCodeURL("")
	log.Infof(c, "Visit the URL for the auth dialog: %v", url)

	// Redirect user to that page.
	http.Redirect(w, r, url, http.StatusFound)
	log.Infof(c, "Leaving handleAuthorize")
}

//
func handleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	// Initialize an appengine context.
	c := appengine.NewContext(r)
	log.Infof(c, "In handleOAuth2Callback")

	// log.Infof(c, "r: %v", r)
	code := r.FormValue("code")
	// log.Infof(c, "Code: %v", code)

	tok, err := conf.Exchange(c, code)
	if err != nil {
		log.Errorf(c, "%v", err)
	}
	// log.Infof(c, "Token: %v", tok)

	client := conf.Client(c, tok)
	// log.Infof(c, "Client: %v", client)

	// PLUS SERVICE CLIENT
	pc, err := plus.New(client)
	if err != nil {
		log.Errorf(c, "An error occurred creating Plus client: %v", err)
	}

	person, err := pc.People.Get("me").Do()
	if err != nil {
		log.Errorf(c, "Person Error: %v", err)
	}
	log.Infof(c, "Name: %v", person.DisplayName)

	// DRIVE CLIENT
	dc, err := drive.New(client)
	if err != nil {
		log.Errorf(c, "An error occurred creating Drive client: %v", err)
	}
	// log.Infof(c, "Files: %v", dc.Files.List().Do())
	files, err := dc.Files.List().Q("title contains 'Libra Database:'").Do()
	filenames := make([]string, len(files.Items))
	for i, value := range files.Items {
		// log.Infof(c, "Files [%v]: %v", key, value.OriginalFilename)
		log.Infof(c, "Files: %v", value.Title)
		// filenames[i] = fmt.Sprintf("<option>%v</option>\n", value.Title)
		filenames[i] = value.Title
	}

	data := struct {
		DisplayName string
		LibraFiles  []string
	}{
		DisplayName: person.DisplayName,
		// LibraFiles:  strings.Join(filenames, ""),
		LibraFiles: filenames,
	}

	// Render the user's information.
	err = cached_templates.ExecuteTemplate(w, "userInfo.html", data)
}
