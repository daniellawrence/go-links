package main

import (
	"log"
	"net/http"
	"strings"
	"html/template"
	"path/filepath"
	"os"

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
	GoLinks []GoLink
	GoLink  GoLink
}


var db *gorm.DB


func (g GoLink) HTTPRedirect(w http.ResponseWriter, r *http.Request, args string) {
	log.Printf("redirecting %#v", g)
	http.Redirect(w, r, g.Target, 303)
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

	log.Printf("path: %v", path)
	log.Printf("name=%s", name)
	log.Printf("args=%s", args)

	g := GoLink{}
	db.Where("name = ?", name).First(&g)

	// If the database look was sucessful, the redirect
	if g.Name == name && name != "" {
		g.HTTPRedirect(w, r, args)
		return
	}

	if path == "/" {
		log.Print("index.html")
		serveTemplate(w, r, "index.html")
		return
	}

	if path == "/ping" {
		log.Print("ping")
		w.Write([]byte("pong"))
		return
	}

	log.Print("missing")
	serveTemplate(w, r, "missing.html")
	return
}


func serveTemplate(w http.ResponseWriter, r *http.Request, path string) {
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

	log.Printf("template OK: %v, %v", path, info)
	tmpl.ExecuteTemplate(w, path, nil)
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
