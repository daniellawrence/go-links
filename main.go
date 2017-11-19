package main

import (
	"fmt"
	"html/template"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type GoLink struct {
	gorm.Model
	Owner     string
	Name      string
	Target    string
	ViewCount int
}

type PageContext struct {
	TopGoLinks []GoLink
	GoLink     GoLink
}

var db *gorm.DB

func (g GoLink) HTTPRedirect(w http.ResponseWriter, r *http.Request, args string) {
	log.Printf("redirecting %#v", g)
	http.Redirect(w, r, g.Target, 303)
}

func (g GoLink) HTTPRedirectToView(w http.ResponseWriter, r *http.Request) {
	target := fmt.Sprintf("/golink/%d/", g.ID)
	http.Redirect(w, r, target, 303)
}

func CreateNewGoLink(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	target := r.FormValue("target")
	log.Printf("NEW: %s,%s\n", name, target)

	g, _ := GoLinkFromName(name)

	if g.Name == "" {
		// If its a new record (no match), then create a new one.
		g = GoLink{Owner: "dlawrenc", Name: name, Target: target}
		db.NewRecord(g)
		db.Create(&g)

	} else {
		// If its a found record, then update it
		g.Name = name
		g.Target = target
		db.Save(&g)
	}

	g.HTTPRedirectToView(w, r)

}

func GoLinkFromName(name string) (g GoLink, err error) {
	db.Where("name = ?", name).First(&g)
	return g, nil
}

func GoLinkFromID(id int) (g GoLink, err error) {
	db.Where("id = ?", id).First(&g)
	return g, nil
}

func ParseInboundPath(p string) (name string, args string) {
	path := strings.Replace(p, "%20", " ", -1)
	pathSlice := strings.Split(path, " ")
	name = strings.Trim(pathSlice[0], "/")
	args = strings.Join(pathSlice[1:], " ")
	return name, args
}

func route(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	name, args := ParseInboundPath(path)

	if path == "/new/" {
		CreateNewGoLink(w, r)
		return
	}

	g, _ := GoLinkFromName(name)

	// If the database look was sucessful, the redirect
	if g.Name == name && name != "" {
		g.HTTPRedirect(w, r, args)
		return
	}

	if path == "/" {
		log.Print("index.html")
		serveTemplate(w, r, "index.html", g)
		return
	}

	if path == "/ping" {
		log.Print("ping")
		w.Write([]byte("pong"))
		return
	}

	viewRE := regexp.MustCompile("/golink/([0-9]+)/")
	editRE := regexp.MustCompile("/golink/([0-9]+)/edit/")

	viewResponse := viewRE.FindSubmatch([]byte(path))
	if viewResponse == nil {
		log.Printf("BAD")
		return
	}

	idBig := new(big.Int)
	idBig.SetString(string(viewResponse[1]), 2)
	id := idBig.Sign()

	if viewRE.MatchString(path) {
		g, _ = GoLinkFromID(id)
		log.Printf("view - %s - %d - %#v\n", path, id, g)
		serveTemplate(w, r, "view.html", g)
		return
	}

	if editRE.MatchString(path) {
		g, _ = GoLinkFromID(id)
		log.Printf("edit - %s - %d - %#v\n", path, id, g)
		serveTemplate(w, r, "edit.html", g)
		return
	}

	log.Print("index.html")
	serveTemplate(w, r, "index.html", g)
	return
}

func serveTemplate(w http.ResponseWriter, r *http.Request, path string, golink GoLink) {
	lp := filepath.Join("templates", "layout.html")
	fp := filepath.Join("templates", filepath.Clean(path))

	info, err := os.Stat(fp)

	if err != nil {
		log.Fatalf("missing template: %v", err)

		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
	}

	if info.IsDir() {
		log.Fatalf("missing DIR: %v", err)
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFiles(lp, fp)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, http.StatusText(500), 500)
		return
	}

	p := PageContext{
		TopGoLinks: FetchTopGoLinks(20),
		GoLink:     golink,
	}

	log.Printf("template OK: %v, %v", path)
	err = tmpl.ExecuteTemplate(w, "layout", p)
	if err != nil {
		log.Println(err.Error())
	}

	err = tmpl.ExecuteTemplate(w, "body", p)
	if err != nil {
		log.Println(err.Error())
	}

}

func FetchTopGoLinks(count int) (topGoLinks []GoLink) {
	topGoLinks = []GoLink{}
	db.Find(&topGoLinks).Order("created_at").Limit(20)
	return topGoLinks

}

func main() {
	db, _ = gorm.Open("sqlite3", "/tmp/test.db")
	db.LogMode(true)
	db.SingularTable(true)
	db.AutoMigrate(&GoLink{})

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	wiki := GoLink{Owner: "dlawrenc", Name: "wiki", Target: "http://wiki/corp"}
	db.NewRecord(wiki)
	db.Create(&wiki)

	http.HandleFunc("/", route)

	log.Println("Listening on :8080")
	http.ListenAndServe(":8080", nil)

}
