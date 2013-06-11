package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"log/syslog"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"
)

var logger, _ = syslog.New(syslog.LOG_NOTICE, "go-gyazo-server")

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":80", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		if r.Method == "POST" {
			Upload(w, r)
		} else {
			TopPage(w, r)
		}
	} else {
		Image(w, r)
	}
}

func TopPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://github.com/dddaisuke/go-gyazo-server", 301)
}

func Image(w http.ResponseWriter, r *http.Request) {
	_, id := path.Split(r.URL.Path)
	file_path := fmt.Sprintf("images/%s/%s/%s/%s", id[0:2], id[2:4], id[4:6], id[6:])
	body, err := ioutil.ReadFile(file_path)
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(body)
}

func Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "invalid request", 500)
		return
	}

	ct := r.Header.Get("Content-Type")
	if strings.SplitN(ct, ";", 2)[0] != "multipart/form-data" {
		http.Error(w, "invalid request", 500)
		return
	}

	_, params, err := mime.ParseMediaType(ct)
	if err != nil {
		http.Error(w, "invalid request", 500)
		return
	}

	boundary, ok := params["boundary"]
	if !ok {
		http.Error(w, "invalid request", 500)
		return
	}

	reader := multipart.NewReader(r.Body, boundary)
	var image []byte
	for {
		part, err := reader.NextPart()
		if part == nil || err != nil {
			break
		}

		if part.FormName() != "imagedata" {
			continue
		}
		v := part.Header.Get("Content-Disposition")

		if v == "" {
			continue
		}
		d, _, err := mime.ParseMediaType(v)

		if err != nil {
			continue
		}

		if d != "form-data" {
			continue
		}
		image, _ = ioutil.ReadAll(part)
	}

	sha := sha1.New()
	sha.Write(image)
	id := fmt.Sprintf("%x", string(sha.Sum(nil))[0:8])
	file_path := fmt.Sprintf("images/%s/%s/%s", id[0:2], id[2:4], id[4:6])
	file_name := fmt.Sprintf("%s", id[6:])
	os.MkdirAll(file_path, 0700)
	ioutil.WriteFile(file_path+"/"+file_name+".png", image, 0600)

	host := r.Host
	hi := strings.SplitN(host, ":", 2)
	if len(hi) == 2 && hi[1] == "80" {
		host = hi[0]
	}
	logger.Notice("http://" + host + "/" + id + ".png")
	w.Write([]byte("http://" + host + "/" + id + ".png"))
}
