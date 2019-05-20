package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/microcosm-cc/bluemonday"
	"gopkg.in/russross/blackfriday.v2"
)

// API domain
var API string

func init() {
	isDev := os.Getenv("DEV_MODE") == "true"

	if isDev {
		API = os.Getenv("API_URL")
	} else {
		// Docker networking will  take care of that (docker-compose, or aws)
		API = "http://api:8080"
	}
}

// Blog type - In data fetched from API
type Blog struct {
	ID              string    `json:"_id,omitempty"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	DescriptionLink string    `json:"description_link"`
	Author          string    `json:"author"`
	FormattedDate   string    `json:"formatted_date"`
	HTML            string    `json:"html"`
	Markdown        string    `json:"markdown"`
	DocType         string    `json:"doc_type"`
	Thumbnail       string    `json:"thumbnail"`
	IsTechnical     bool      `json:"is_technical"`
	IsPublic        bool      `json:"is_public"`
	IsDeleted       bool      `json:"is_deleted"`
	IsSeries        bool      `bson:"is_series" json:"is_series"`
	SubBlogs        []SubBlog `bson:"sub_blogs" json:"sub_blogs"`
}

// SubBlog - Blog model type
type SubBlog struct {
	Title         string `bson:"title" json:"title"`
	Description   string `bson:"description" json:"description"`
	FormattedDate string `bson:"formatted_date" json:"formatted_date"`
	HTML          string `bson:"html" json:"html"`
	Markdown      string `bson:"markdown" json:"markdown"`
	DocType       string `bson:"doc_type" json:"doc_type"`
}

// FetchDataAsync asynchronously fetches data from the provided url writes
// the response data into the provided channel, if there's
// any network error, it passes <nil> into the channel
func FetchDataAsync(url string, c chan []byte) {
	// Fetching data from the siven URL
	resp, err := http.Get(url)
	if err != nil {
		c <- nil
		return
	}
	defer resp.Body.Close()

	// Reading the response body
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c <- nil
		return
	}

	c <- b // Passing data into the echannels
}

// FetchDataSync synchronously fetches data
func FetchDataSync(url string) ([]byte, error) {
	// Fetching data from the siven URL
	resp, err := http.Get(url)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	// return the body, and parser error
	return ioutil.ReadAll(resp.Body)
}

// EachThreadHandler reads thread (blog) data from teh api
// and fetches the corrosponding document at the given index from
// web, this handler doesn't include cahcing in the api side, mainly
// because cashing can cause some issues while index change in sub_blog
func EachThreadHandler(w http.ResponseWriter, r *http.Request) {
	bIDStr := mux.Vars(r)["id"]                            // Blog (thread) ObjectID
	index, err := strconv.Atoi(r.URL.Query().Get("index")) // Reading provided index
	if err != nil || index < 0 {
		RenderError(w, 400, "Invalid Index")
		return
	}

	// Read blog data from the API
	b1, err := FetchDataSync(API + "/api/public/blog/" + bIDStr)
	if err != nil {
		RenderError(w, 404, "Blog Not Found")
		return
	}

	// Parsing api data into blog struct
	var data Blog
	err = json.Unmarshal(b1, &data)
	if err != nil {
		RenderError(w, 500, "Internal Server Error")
		return
	}

	// If not a Thread
	if !data.IsSeries {
		RenderError(w, 400, "Not A Thread")
		return
	}

	// If thread doesn't include any subblog
	if len(data.SubBlogs) == 0 {
		RenderError(w, 400, "Empty Thread")
		return
	}

	// If index overflow
	if index+1 > len(data.SubBlogs) {
		http.Redirect(w, r, "/thread/"+bIDStr+"?index=0", http.StatusSeeOther)
		return
	}

	var docSrc string                                  // Source document URL
	isMd := data.SubBlogs[index].DocType == "markdown" // If doctype is Markdown or not
	if isMd {
		docSrc = data.SubBlogs[index].Markdown
	} else {
		docSrc = data.SubBlogs[index].HTML
	}

	// Fetching source document
	b2, err := FetchDataSync(docSrc)
	if err != nil {
		RenderError(w, 404, "Document Not Found")
		return
	}

	// Unsafe HTML (From Doc)
	var unsafe []byte
	if isMd {
		// Generating HTML from Markdown
		unsafe = blackfriday.Run(b2)
	} else {
		unsafe = b2
	}

	// Document HTML
	html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)

	// Parsing templates
	t, err := template.ParseFiles(
		"static/pages/each-thread.html",
		"static/partials/thread-nav.html",
		"static/partials/header.html",
	)
	if err != nil {
		fmt.Println("Bazinga:", err)
		RenderError(w, 500, "Internal Server Error")
		return
	}

	var prevURL string
	var nextURL string

	if index == 0 {
		prevURL = ""
	} else {
		prevURL = "/thread/" + data.ID + "?index=" + strconv.Itoa(index-1)
	}

	if index+1 < len(data.SubBlogs) {
		nextURL = "/thread/" + data.ID + "?index=" + strconv.Itoa(index+1)
	} else {
		nextURL = ""
	}
	// Executing Template
	err = t.Execute(w, struct {
		Data       Blog
		SubBlog    SubBlog
		Index      int
		PrevSubURL string
		NextSubURL string
		HTML       string
		Project    bool
	}{
		Data:       data,
		SubBlog:    data.SubBlogs[index],
		Index:      index,
		PrevSubURL: prevURL,
		NextSubURL: nextURL,
		HTML:       fmt.Sprintf("%s\n", html),
		Project:    false,
	})

	if err != nil {
		fmt.Println("Error:", err)
		WriteError(w, 500, err, err.Error())
	}
}

// EachBlogHandler - Fetches single Blog data and Document for that Blog,
// Renders document (Html or Markdown) inside HTML template
func EachBlogHandler(w http.ResponseWriter, r *http.Request) {
	bIDStr := mux.Vars(r)["id"] // Blog ObjectId String

	c1 := make(chan []byte) // Channel for Data Fetching
	c2 := make(chan []byte) // Channel for Document Fetching

	// Get Public Data from API
	go FetchDataAsync(API+"/api/public/blog/"+bIDStr, c1)

	// Get Public Data from API
	go FetchDataAsync(API+"/api/public/blog/doc/"+bIDStr, c2)

	b1 := <-c1 // Blog data 1 ([]byte)
	b2 := <-c2 // Document data 2 ([]byte)

	// Check Error
	if b1 == nil || b2 == nil {
		RenderError(w, 404, "Blog Not Found")
		return
	}

	// Unmarshaling Body Data
	var data Blog

	err := json.Unmarshal(b1, &data)
	if err != nil {
		RenderError(w, 500, "Internal Server Error")
		return
	}

	// Unsafe HTML (From Doc)
	var unsafe []byte

	if data.DocType == "markdown" {
		// Generating HTML from Markdown
		unsafe = blackfriday.Run(b2)
	} else {
		unsafe = b2
	}

	// Document HTML
	html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)

	// Parsing templates
	t, err := template.ParseFiles(
		"static/pages/each-doc.html",
		"static/partials/header.html",
	)
	if err != nil {
		RenderError(w, 500, "Internal Server Error")
		return
	}

	// Executing Template
	err = t.Execute(w, struct {
		Data    Blog
		HTML    string
		Project bool
	}{
		Data:    data,
		HTML:    fmt.Sprintf("%s\n", html),
		Project: false,
	})

	if err != nil {
		WriteError(w, 500, err, err.Error())
	}
}

// BlogsHandler - Handler for All Blogs Page
func BlogsHandler(w http.ResponseWriter, r *http.Request) {
	// channel
	c := make(chan []byte)
	// var blogsroute string
	var tech bool
	var cookie http.Cookie

	// Checking if technical only
	qTech := r.URL.Query()["tech"]

	// Checking if query string exist or not
	if len(qTech) > 0 {
		if qTech[0] == "true" {
			tech = true
			cookie = http.Cookie{Name: "tech", Value: "true", Path: "/", Expires: time.Now().AddDate(0, 1, 0), MaxAge: 86400} // Cookie
		} else if qTech[0] == "false" {
			cookie = http.Cookie{Name: "tech", Value: "false", Path: "/", Expires: time.Now().AddDate(0, 1, 0), MaxAge: 86400} // Cookie
		}
	} else {
		// If nothing in query string read read from cookie
		var tCuki, err = r.Cookie("tech")
		// If value exists and its "true", then make tech true
		if err == nil && tCuki.Value == "true" {
			tech = true
		}
	}

	// Get Public Data from API
	if tech {
		go FetchDataAsync(API+"/api/public/blog/all?tech=true", c)
	} else {
		go FetchDataAsync(API+"/api/public/blog/all?tech=false", c)
	}

	// Unmarshaling Body Data
	var data []Blog

	err := json.Unmarshal(<-c, &data)
	if err != nil {
		RenderError(w, 500, "Internal Server Error")
		return
	}

	// Parsing templates
	t, err := template.ParseFiles(
		"static/pages/blogs.html",
		"static/partials/header.html",
		"static/partials/blogs-item.html",
		"static/partials/social-btns.html",
	)
	if err != nil {
		RenderError(w, 500, "Internal Server Error")
	}

	http.SetCookie(w, &cookie)

	// Executing Template
	err = t.Execute(w, struct {
		Tech bool
		Data []Blog
	}{
		Tech: tech,
		Data: data,
	})

	if err != nil {
		WriteError(w, 500, err, err.Error())
	}
}
