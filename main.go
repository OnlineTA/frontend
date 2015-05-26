package main

import (
  "net/http"
  "html/template"
  "os"
  "os/user"
  "io"
  "log"
  "path"
  "strconv"
  //"mime/multipart"
  "github.com/gorilla/mux"
  //"gopkg.in/yaml.v2"
  "code.google.com/p/go-uuid/uuid"
  "github.com/onlineta/common"
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

var base_dir = "/tmp/onlineta/"
var submission_subdir = "submissions/"
var submit_dir_full = base_dir + submission_subdir

var gconfig *common.Config

func fail500(w http.ResponseWriter, err error) {
  http.Error(w, err.Error(), http.StatusInternalServerError)
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


func receive_submission(w http.ResponseWriter, r *http.Request) {
  // TODO: Add defer statement here calling a function for rolling
  // back changes made in case this function returns prematurely
  // Could be implemented by pushing each file operation performed
  // to a call stack which is cleared upon successful exit and
  // and upon premature exit the inverse of each of its containing
  // executed

  // Extract meta data from
  vars := mux.Vars(r)

  //unknown_file := false

  meta := common.Metadata{}
  //meta := common.Metadata{} {
  //  vars["course"].
  //}
  meta.Course = vars["course"]
  meta.Assignment = vars["assignment"]
  if user := vars["user"]; user == "" {
    // Assign anonymous username
  } else {
    meta.User = user
  }
  meta.Id = uuid.New()
  meta.Status = common.STATUS_ACCEPTED

  if err := r.ParseMultipartForm(107374182-4); err != nil {
    fail500(w,err)
    log.Print(err)
    return
  }

  //var file_header multipart.FileHeader
  src, file_header, err := r.FormFile("src")
  if err != nil {
    fail500(w, err)
    log.Print(err)
    return
  }

  // Try to identify the kind of file that we're working with
  // TODO: Do something with thus info
  reader := io.LimitReader(src, 512)
  header := make([]byte, 512);
  reader.Read(header)
  mime := http.DetectContentType(header)
  log.Print("Detected mimetype " + mime)

  //data := struct {meta Metadata, error Error}

  // Save file
  dir := path.Join(common.ConfigValue("SubmissionDir"), meta.Id)
  if err := os.Mkdir(dir, 0700); err != nil {
    log.Print(err)
    fail500(w, err)
    return
  }

  upload_name := path.Join(dir, file_header.Filename)
  f, err := os.OpenFile(upload_name, os.O_CREATE|os.O_WRONLY, 0600)
  if err != nil {
    log.Print(err)
    fail500(w, err)
    return
  }
  defer f.Close()

  src.Seek(0, 0)
  if _,err := io.Copy(f, src); err != nil {
    log.Print(err)
    fail500(w, err)
    return
  }

  // FIXME: Race condition between file writing and file chowning
  // could leave behind submissions owned by the onlineTA user

  // Set owner if username is known
  if meta.User != "" {
    // Lookup user
    user, err := user.Lookup(meta.User)
    if err != nil {
      log.Print(err)
      fail500(w,err)
      return
    }
    //TODO: handle error
    uid, _ := strconv.Atoi(user.Uid)
    if err := os.Chown(upload_name, uid, 65534); err != nil {
      log.Print(err)
      fail500(w,err)
      return
    }
    // Set containing directory owner
    if err := os.Chown(dir, uid, 65534); err != nil {
      log.Print(err)
      fail500(w,err)
      return
    }
  }

  // Save metadata
  if err := meta.Commit(); err != nil {
    log.Print(err)
    return
  }

  // TODO: Cleanup saved file and return error to user if saving
  // of metadata file fails for some reason

  templates.ExecuteTemplate(w, "success.html", meta)

  return
}

func query_handler(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)

  meta, err := common.Get(vars["id"])
  if err != nil {
    fail500(w, err)
    log.Print(err)
    return
  }

  log.Print(int(meta.Status))
  data := struct {
    Id string
    Status string
    Assessment string
  }{
    meta.Id,
    meta.Status.Description(),
    "",
  }

  templates.ExecuteTemplate(w, "query.html", data)
}

func assessment_handler(w http.ResponseWriter, r* http.Request) {
  vars := mux.Vars(r)

  meta, err := common.Get(vars["id"])
  if err != nil {
    fail500(w, err)
    log.Print(err)
    return
  }

  // FIXME: Why do we need two calls to start assessment server
  assessments := New()
  assessments.Serve()

  assess_ch, ok := Subscribe(meta.Id)
  if !ok {
    //fail500(w, "")
    log.Print("")
    return
  }

  assessment := <- assess_ch
  if assessment == "" {
    assessment = "Assessment retrieval timed out, Please try again"
  }

  templates.ExecuteTemplate(w, "submission.html", assessment)
}

func init_env(){
  log.Print(gconfig.Default.Basedir)
  if err := os.Mkdir(gconfig.Default.Basedir, 0700); !os.IsExist(err)  && err != nil {
    panic("Failed ot create directory " + gconfig.Default.Basedir)
  }
  if err := os.Mkdir(gconfig.Default.SubmissionDir, 0700); !os.IsExist(err) && err != nil {
    panic("Failed to create directory")
  }
  if err := os.Mkdir(gconfig.Default.IncomingDir, 0700); !os.IsExist(err) && err != nil {
    panic("Failed to create directory")
  }
}

func main() {
  gconfig = new(common.Config)
  if err := gconfig.Parse("../onlineta.conf"); err != nil {
    log.Print(err)
    return
  }
  log.Print(gconfig.Default.Basedir)

  // Make config lookups available through the ConfigCh channel
  gconfig.Serve()

  init_env()

  router := mux.NewRouter()
  router.HandleFunc("/", show_index).Methods("GET")
  router.HandleFunc("/submit/{course}/{assignment}/{user:[a-z]{3}[0-9]{3}}", named_submit_handler).Methods("GET")
  router.HandleFunc("/submit/{course}/{assignment}", anonymous_submit_handler).Methods("GET");
  router.HandleFunc("/submit/{course}/{assignment}/{user:[a-z]{3}[0-9]{3}}", receive_submission).Methods("POST", "PUT")
  router.HandleFunc("/submit/{course}/{assignment}", receive_submission).Methods("POST", "PUT");
  // TODO: Check that ID is a valid UUID
  router.HandleFunc("/query/{id}", query_handler).Methods("GET")
  router.HandleFunc("/assessment/{id}", query_handler).Methods("GET")

  http.Handle("/", router)
  http.ListenAndServe(":8080", nil)

}
