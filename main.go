package main

import (
  "net/http"
  "html/template"
  "os"
  "io"
  "log"
  "github.com/gorilla/mux"
  //"gopkg.in/yaml.v2"
  "code.google.com/p/go-uuid/uuid"
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

var base_dir = "/tmp/onlineta/"
var submission_subdir = "submissions/"
var submit_dir_full = base_dir + submission_subdir

func fail500(w http.ResponseWriter, err error) {
  http.Error(w, err.Error(), http.StatusInternalServerError)
}

func handler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
  case "GET":
    log.Print("foo");
  case "POST":

    defer src.Close()
    err = os.Mkdir("/tmp/onlineta", 0777)
    if err != nil {
      fail500(w, err)
      log.Print(err)
      return
    }
    f, err := os.Create("/tmp/onlineta/foo");
    defer f.Close()

    if _,err := io.Copy(f, src); err != nil {
      fail500(w, err)
      return
    }
  }
}

func show_index(w http.ResponseWriter, r *http.Request) {
  templates.ExecuteTemplate(w, "index.html", nil)
}

func anonymous_submit_handler(w http.ResponseWriter, r *http.Request) {
  log.Print("Anon submit")
  templates.ExecuteTemplate(w, "upload.html", nil)
}

func named_submit_handler(w http.ResponseWriter, r *http.Request) {
  log.Print("I know who you are submit")
  templates.ExecuteTemplate(w, "upload.html", nil)
}

type Metadata struct {
  Id string
  Course string
  Assignment string
  User string
}

func receive_submission(w http.ResponseWriter, r *http.Request) {
  // Extract meta data from

  vars := mux.Vars(r)

  unknown_file := false

  meta := Metadata{}
  meta.Course = vars["course"]
  meta.Assignment = vars["assignment"]
  meta.User = vars["anon"]
  meta.Id = uuid.New()

  err := r.ParseMultipartForm(524288)
  //m := r.MultipartForm

  if src, _, err := r.FormFile("src"); err != nil {
      fail500(w, err)
      log.Fatal(err)
      return
  }

  // Try to identify the kind of file that we're working with
  fheader := io.LimitedReader(src, 512)
  mime := http.DetectContentType(reader.Read())
  log("Detected mimetype " + mime)

  // Handle compressed archives
  // TODO: Get list of accepted file formats from assignment specification
 / switch {
  case mime == "application/x-gzip":
    // Extract gzip and check resultint tar
  case mime == "applicatin/zip":
    // Extract zip file
  default:
    unknown_file = true
  }

  data := struct {meta Metadata, error Error}

  templates.ExecuteTemplate(w, "success.html", meta)

  return
}

func init_env(){
  if err := os.Mkdir(base_dir, 0700); !os.IsExist(err)  && err != nil {
    panic("Failed ot create directory")
  }
  if err := os.Mkdir(submit_dir_full, 0700); !os.IsExist(err) && err != nil {
    panic("Failed to create directory")
  }
}
func main() {

  init_env()

  router := mux.NewRouter()
  router.HandleFunc("/", show_index).Methods("GET")
  router.HandleFunc("/submit/{course}/{assignment}", anonymous_submit_handler).Methods("GET");
  router.HandleFunc("/submit/{course}/{assignment}", receive_submission).Methods("POST", "PUT");
  router.HandleFunc("/submit/{course}/{assignment}/{user}", named_submit_handler).Methods("POST", "PUT")

  http.Handle("/", router)
  http.ListenAndServe(":8080", nil)

}
