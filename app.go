package mylibraweight

import (
	"bufio"
	"code.google.com/p/gorilla/sessions"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jws"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/plus/v1"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"
)

// Cache all of the HTML files in the templates directory so that we only have to hit disk once.
var cached_templates = template.Must(template.ParseGlob("templates/*.html"))

// Cookie store
var store sessions.Store

type MyGoogleUser struct {
	Name  string
	Email string
	Token *oauth2.Token
	Set   bool
}

var MyUser = MyGoogleUser{
	Name:  "",
	Email: "",
	Token: nil,
	Set:   false,
}

// App Engine instance
var conf = &oauth2.Config{
	ClientID:     "285312328170-a54o8ukf7lmlan610vfh1cr4iq4boemp.apps.googleusercontent.com", // from https://console.developers.google.com/project/<your-project-id>/apiui/credential
	ClientSecret: "IAFj6KxoAyYRKGYHiPK4I88Z",
	RedirectURL:  "http://mylibraweight.appspot.com/oauth2callback",
	Scopes: []string{
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/userinfo.profile",
		"https://www.googleapis.com/auth/userinfo.email",
	},
	Endpoint: google.Endpoint,
}

// App Engine instance
var confFacebook = &oauth2.Config{
	ClientID:     "854784501254889", // 285312328170-c3rn57hq6bphe1id87p3o2ehhq6ru0g9.apps.googleusercontent.com",
	ClientSecret: "4289bb536452ff767b47fe644a6e5af9",
	RedirectURL:  "http://mylibraweight.appspot.com/oauth2callbackFacebook",
	Scopes:       []string{"email"},
	Endpoint:     facebook.Endpoint,
}

// // Debugging on local instance - Run local instance as goapp serve -port 10080
// var conf = &oauth2.Config{
// 	ClientID:     "285312328170-7dvm2p1sa9tnfpfblopuk4eqp0r80jvl.apps.googleusercontent.com",
// 	ClientSecret: "-qW7bzzoddgeXOIo-G-4H_4K",
// 	RedirectURL:  "http://localhost:10080/oauth2callback",
// 	Scopes: []string{
// 		"https://www.googleapis.com/auth/drive",
// 		"https://www.googleapis.com/auth/userinfo.profile",
// 		"https://www.googleapis.com/auth/userinfo.email",
// 	},
// 	Endpoint: google.Endpoint,
// }

// // Debugging on local instance
// var confFacebook = &oauth2.Config{
// 	ClientID:     "858462280887111",
// 	ClientSecret: "c168b5d95acf86788491f38232b98eef",
// 	RedirectURL:  "http://localhost:10080/oauth2callbackFacebook",
// 	Scopes:       []string{"email"},
// 	Endpoint:     facebook.Endpoint,
// }

// This is the URL that Google has defined so that an authenticated application may obtain the user's info in json format.
// const profileInfoURL = "https://www.googleapis.com/oauth2/v1/userinfo?alt=json"

// init is called before the application starts.
func init() {
	// Register a handler for /hello URLs.
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/authorize", handleAuthorize)
	http.HandleFunc("/oauth2callback", handleOAuth2Callback)
	http.HandleFunc("/about", handleAbout)
	http.HandleFunc("/contact", handleContact)
	http.HandleFunc("/authorizeFacebook", handleAuthorizeFacebook)
	http.HandleFunc("/oauth2callbackFacebook", handleOAuth2CallbackFacebook)

	// store = sessions.NewCookieStore([]byte(os.Getenv("KEY")))
	store = sessions.NewCookieStore([]byte("MyVeryPrivateString"))
	gob.Register(&MyGoogleUser{})
}

//
func handleRoot(w http.ResponseWriter, r *http.Request) {
	// Initialize an appengine context.
	c := appengine.NewContext(r)

	// Get session
	session, err := store.Get(r, "session-name")
	if err != nil {
		log.Errorf(c, "Session Error in handleRoot: %v (%s)", err.Error(), session.Name())
		return
	}
	// Retrieve our MyGoogleUser struct and type-assert it.  If it is valid, skip the Root page and go to UserInfo page.
	// if mGU, ok := session.Values["user"].(*MyGoogleUser); ok && mGU.Set && mGU.Token.Valid() {
	// 	handleOAuth2Callback(w, r)
	// } else {
	err = cached_templates.ExecuteTemplate(w, "notAuthenticated.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
	// }
}

//
func handleAuthorize(w http.ResponseWriter, r *http.Request) {
	// Initialize an appengine context.
	c := appengine.NewContext(r)
	// log.Infof(c, "In handleAuthorize")

	// Get the Google URL which shows the Authentication page to the user.
	// url := conf.AuthCodeURL("state")
	// url := conf.AuthCodeURL("")
	url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline) // Set such that returns refresh token too.
	log.Infof(c, "Visit the URL for the auth dialog: %v", url)

	// Redirect user to that page.
	http.Redirect(w, r, url, http.StatusFound)
	// log.Infof(c, "Leaving handleAuthorize")
}

//
func handleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	// Initialize an appengine context.
	c := appengine.NewContext(r)
	// log.Infof(c, "In handleOAuth2Callback")

	var client *http.Client = nil

	// Get session
	session, err := store.Get(r, "session-name")
	if err != nil {
		log.Errorf(c, "Session Error in handleOAuth2Callback: %v", err.Error())
		return
	}

	// Retrieve our struct and type-assert it
	if mGU, ok := session.Values["user"].(*MyGoogleUser); ok {
		// log.Infof(c, "MyGoogleUser in handleOAuth2Callback - Email: %v", mGU.Email)
		if mGU.Set {
			// Use the user stored in session
			MyUser = *mGU
			log.Infof(c, "handleOAuth2Callback - Session User: %v (%v)", MyUser.Name, MyUser.Email)
			// Check if token has expired
			if !MyUser.Token.Valid() {
				// Refresh token
				code := r.FormValue("code")
				tok, err := conf.Exchange(c, code)
				if err != nil {
					log.Errorf(c, "%v", err)
				}
				MyUser.Token = tok
				session.Values["user"] = MyUser
				session.Save(r, w)
			}
			client = conf.Client(c, MyUser.Token)
		}
	} else {
		// Login user by getting token from response
		code := r.FormValue("code")
		tok, err := conf.Exchange(c, code)
		if err != nil {
			log.Errorf(c, "%v", err)
		}
		// log.Infof(c, "Token: %v", tok)

		// Email etc. to come from token information
		m := tok.Extra("id_token").(string)
		// log.Infof(c, "M: %v", m)
		cs, err := jws.Decode(m)
		if err != nil {
			log.Errorf(c, "JWS Token decode error: %v", err)
		} else {
			// log.Infof(c, "ClaimSet User #: %v", cs.Sub)
		}

		client = conf.Client(c, tok)
		// log.Infof(c, "Client: %v", client)

		// PLUS SERVICE CLIENT
		pc, err := plus.New(client)
		if err != nil {
			log.Errorf(c, "An error occurred creating Plus client: %v", err)
		}

		// person, err := pc.People.Get("me").Do()
		person, err := pc.People.Get(cs.Sub).Do()
		if err != nil {
			log.Errorf(c, "Person Error: %v", err)
		}
		// log.Infof(c, "Name: %v", person.DisplayName)
		// log.Infof(c, "All Person information: %v", person)
		email := ""
		for _, e := range person.Emails {
			// log.Infof(c, "Emails: [%v] %v", e.Type, e.Value)
			if e.Type == "account" {
				email = e.Value
			}
		}
		// log.Infof(c, "Email: %v", email)

		MyUser.Name = person.DisplayName
		MyUser.Email = email
		MyUser.Token = tok
		MyUser.Set = true

		session.Values["user"] = MyUser
		session.Save(r, w)
	}

	// DRIVE CLIENT
	dc, err := drive.New(client)
	if err != nil {
		log.Errorf(c, "An error occurred creating Drive client: %v", err)
	}

	var filenames []string
	var graphData template.JS

	// log.Infof(c, "Files: %v", dc.Files.List().Do())
	files, err := dc.Files.List().Q("title contains 'Libra Database:'").Do()
	if err != nil {
		log.Errorf(c, "An error occured getting Libra Database files from Drive client: %v", err)
	}
	if err == nil && len(files.Items) > 0 {
		filenames = make([]string, len(files.Items))
		for i, value := range files.Items {
			// log.Infof(c, "Files: %v", value.Title)
			filenames[i] = value.Title
			// filenames[i] = fmt.Sprint(value.Title, value.FileExtension)
		}

		// For now, get the first file
		// TODO: Get selected file
		// Find fileID
		searchString := fmt.Sprintf("title contains '%v'", filenames[0])
		// log.Infof(c, "SearchString: %v", searchString)
		findFile, err := dc.Files.List().Q(searchString).Do()
		// for i, x := range findFile.Items {
		// 	log.Infof(c, "Files [%v]: %v", i, x.Title)
		// }
		f, err := dc.Files.Get(findFile.Items[0].Id).Do()
		if err != nil {
			log.Errorf(c, "File cannot be found.")
		} else {
			// log.Infof(c, "FileName: %v", f.OriginalFilename)
		}
		downloadUrl := f.DownloadUrl
		if downloadUrl == "" {
			// If there is no downloadUrl, there is no body
			fmt.Printf("An error occurred: File is not downloadable")
			return
		}
		req, err := http.NewRequest("GET", downloadUrl, nil)
		if err != nil {
			fmt.Printf("An error occurred: %v\n", err)
			return
		}

		resp, err := client.Transport.RoundTrip(req)
		// Make sure we close the Body later
		defer resp.Body.Close()
		if err != nil {
			fmt.Printf("An error occurred: %v\n", err)
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("An error occurred: %v\n", err)
			return
		}

		graphData = template.JS(formatDataFromString(c, string(body)))
	} else {
		// No items
		filenames = make([]string, 1, 1)
		graphData = template.JS(",")
	}

	// User data for display on the webpage
	data := struct {
		DisplayName string
		Email       string
		LibraFiles  []string
		GraphData   template.JS
	}{
		DisplayName: MyUser.Name,
		Email:       MyUser.Email,
		LibraFiles:  filenames,
		GraphData:   graphData,
	}

	// Render the user's information.
	err = cached_templates.ExecuteTemplate(w, "userInfo.html", data)
}

// Function to handle drawing the graph on the autherized screen
func formatDataFromString(c context.Context, s string) string {
	r := bufio.NewReader(strings.NewReader(s))
	out := ""
	line, err := r.ReadString('\n')
	for err == nil {
		out = fmt.Sprint(out, formatSingleDataLine(c, line))
		line, err = r.ReadString('\n')
	}
	return out
	// if err != os.EOF {
	//     fmt.Println(err)
	//     return
	// }
}

func formatSingleDataLine(c context.Context, s string) string {
	// Check for comment or blank line
	if s != "" && s[0] != '#' && s[0] != ' ' {
		ss := strings.Split(s, ";")

		if len(ss) > 2 {
			// Manual formatting of data from Libra Database file into Google Chart data line format.
			return fmt.Sprintf("[new Date(%v, %v, %v, %v, %v, %v), %v, %v],\n", s[:4], s[5:7], s[8:10], s[11:13], s[14:16], s[17:19], ss[1], ss[2])
		}
	}
	// Return empty string for comment or blank line
	return ""
}

func handleAbout(w http.ResponseWriter, r *http.Request) {

	err := cached_templates.ExecuteTemplate(w, "about.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}

}

func handleContact(w http.ResponseWriter, r *http.Request) {

	err := cached_templates.ExecuteTemplate(w, "contact.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}

}

//
func handleAuthorizeFacebook(w http.ResponseWriter, r *http.Request) {
	// Initialize an appengine context.
	c := appengine.NewContext(r)
	log.Infof(c, "In handleAuthorizeFacebook")

	// Get the Google URL which shows the Authentication page to the user.
	url := confFacebook.AuthCodeURL("state", oauth2.AccessTypeOffline) // Set such that returns refresh token too.
	log.Infof(c, "Visit the Facebook URL for the auth dialog: %v", url)

	// Redirect user to that page.
	http.Redirect(w, r, url, http.StatusFound)
	log.Infof(c, "Leaving handleAuthorizeFacebook")
}

func handleOAuth2CallbackFacebook(w http.ResponseWriter, r *http.Request) {
	// Initialize an appengine context.
	c := appengine.NewContext(r)
	// log.Infof(c, "In handleOAuth2CallbackFacebook")

	var client *http.Client = nil

	code := r.FormValue("code")
	// log.Infof(c, "Code: %v", code)

	tok, err := confFacebook.Exchange(c, code)
	if err != nil {
		log.Errorf(c, "%v", err)
	}
	log.Infof(c, "Facebook Token: %v", tok)

	// // WHAT TO GO HERE?!  Email etc. to come from token information
	// m := tok.Extra("id_token").(string)
	// log.Infof(c, "M: %v", m)
	// cs, err := jws.Decode(m)
	// if err != nil {
	// 	log.Errorf(c, "JWS Token decode error (Facebook): %v", err)
	// } else {
	// 	log.Infof(c, "Facebook Sub: %v", cs.Sub)
	// 	log.Infof(c, "Facebook Iss: %v", cs.Iss)
	// 	log.Infof(c, "Facebook Aud: %v", cs.Aud)
	// 	log.Infof(c, "Facebook Prn: %v", cs.Prn)
	// 	log.Infof(c, "Facebook Scope: %v", cs.Scope)
	// 	log.Infof(c, "Facebook Typ: %v", cs.Typ)
	// }

	client = confFacebook.Client(c, tok)
	// log.Infof(c, "Client: %v", client)

	response, err := client.Get("https://graph.facebook.com/me?access_token=" + tok.AccessToken)
	if err != nil {
		log.Errorf(c, "Error from Facebook authentication: %v", err)
	}

	str := readHttpBody(response)
	log.Infof(c, "Facebook Response: %v", str)

	// buf := make([]byte, 1024)
	// responseLen, _ := response.Body.Read(buf)

	// buf = buf[:responseLen]

	res := FacebookResponse{}
	err = json.Unmarshal([]byte(str), &res)
	if err != nil {
		log.Errorf(c, "Error from Facebook Unmarshalling: %v", err)
	}

	data := struct {
		DisplayName string
		Email       string
	}{
		DisplayName: fmt.Sprint(res.FirstName, " ", res.LastName),
		Email:       res.Email,
	}

	err = cached_templates.ExecuteTemplate(w, "faceAuth.html", data)
}

// https://www.socketloop.com/tutorials/golang-login-authenticate-with-facebook-example

type FacebookResponse struct {
	Id            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Link          string `json:"link"`
	Picture       string `json:"picture"`
	Gender        string `json:"gender"`
}

func readHttpBody(response *http.Response) string {

	// fmt.Println("Reading body")

	bodyBuffer := make([]byte, 5000)
	var str string

	count, err := response.Body.Read(bodyBuffer)

	for ; count > 0; count, err = response.Body.Read(bodyBuffer) {

		if err != nil {

		}

		str += string(bodyBuffer[:count])
	}

	return str

}
