package mylibraweight

import (
	"fmt"
	"net/http"

	"golang.org/x/oauth2"

	// "google.golang.org/api/drive/v2"
)

// init is called before the application starts.
func init() {
	// Register a handler for /hello URLs.
	http.HandleFunc("/", hello)
}

// hello is an HTTP handler that prints "Hello Gopher!"
func hello(w http.ResponseWriter, r *http.Request) {
	var config = &oauth2.Config{
		ClientID:     "285312328170-a54o8ukf7lmlan610vfh1cr4iq4boemp.apps.googleusercontent.com", // from https://console.developers.google.com/project/<your-project-id>/apiui/credential
		ClientSecret: "IAFj6KxoAyYRKGYHiPK4I88Z",                                                 // from https://console.developers.google.com/project/<your-project-id>/apiui/credential
		Endpoint:     google.Endpoint,
		Scopes:       []string{urlshortener.UrlshortenerScope},
	}

	fmt.Fprint(w, "Hello, Gopher!")
}
