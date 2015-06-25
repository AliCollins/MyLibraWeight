package mylibraweight

import (
	"bufio"
	"fmt"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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

// type MyGoogleUser struct {
// 	Name  string
// 	Email string
// 	// token *oauth2.Token
// }

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
	// RedirectURL: "http://localhost:10080/oauth2callback",
	Scopes: []string{
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/userinfo.profile",
		"https://www.googleapis.com/auth/userinfo.email",
	},
	Endpoint: google.Endpoint,
}

// // Debugging on local instance
// var conf = &oauth2.Config{
// 	ClientID:     "285312328170-7dvm2p1sa9tnfpfblopuk4eqp0r80jvl.apps.googleusercontent.com", // from https://console.developers.google.com/project/<your-project-id>/apiui/credential
// 	ClientSecret: "-qW7bzzoddgeXOIo-G-4H_4K",
// 	RedirectURL:  "http://localhost:10080/oauth2callback",
// 	// RedirectURL: "http://localhost:10080/oauth2callback",
// 	Scopes: []string{
// 		"https://www.googleapis.com/auth/drive",
// 		"https://www.googleapis.com/auth/userinfo.profile",
// 		"https://www.googleapis.com/auth/userinfo.email",
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
	// c := appengine.NewContext(r)
	// log.Infof(c, "In handleAuthorize")

	// Get the Google URL which shows the Authentication page to the user.
	// url := conf.AuthCodeURL("state")
	url := conf.AuthCodeURL("")
	// log.Infof(c, "Visit the URL for the auth dialog: %v", url)

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

	if MyUser.Set {
		// already authorized
		client = conf.Client(c, MyUser.Token)

	} else {
		// log.Infof(c, "r: %v", r)
		code := r.FormValue("code")
		// log.Infof(c, "Code: %v", code)

		tok, err := conf.Exchange(c, code)
		if err != nil {
			log.Errorf(c, "%v", err)
		}
		// log.Infof(c, "Token: %v", tok)

		client = conf.Client(c, tok)
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
		// log.Infof(c, "All Person information: %v", person)
		email := ""
		for _, e := range person.Emails {
			log.Infof(c, "Emails: [%v] %v", e.Type, e.Value)
			if e.Type == "account" {
				email = e.Value
			}
		}
		log.Infof(c, "Email: %v", email)

		MyUser.Name = person.DisplayName
		MyUser.Email = email
		MyUser.Token = tok
		MyUser.Set = true
	}

	// user := MyGoogleUser{person.DisplayName, email, tok}
	// log.Infof(c, "User: %v", user)

	// DRIVE CLIENT
	dc, err := drive.New(client)
	if err != nil {
		log.Errorf(c, "An error occurred creating Drive client: %v", err)
	}
	// log.Infof(c, "Files: %v", dc.Files.List().Do())
	files, err := dc.Files.List().Q("title contains 'Libra Database:'").Do()
	filenames := make([]string, len(files.Items))
	for i, value := range files.Items {
		// log.Infof(c, "Files: %v", value.Title)
		filenames[i] = value.Title
		// filenames[i] = fmt.Sprint(value.Title, value.FileExtension)
	}

	// For now, get the first file
	// TODO: Get selected file
	// Find fileID
	searchString := fmt.Sprintf("title contains '%v'", filenames[0])
	log.Infof(c, "SearchString: %v", searchString)
	findFile, err := dc.Files.List().Q(searchString).Do()
	for i, x := range findFile.Items {
		log.Infof(c, "Files [%v]: %v", i, x.Title)
	}
	f, err := dc.Files.Get(findFile.Items[0].Id).Do()
	if err != nil {
		log.Errorf(c, "File cannot be found.")
	} else {
		log.Infof(c, "FileName: %v", f.OriginalFilename)
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
	// TODO: Change this to do something with the file data!!
	// log.Infof(c, string(body))
	graphData := template.JS(formatDataFromString(c, string(body)))

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
