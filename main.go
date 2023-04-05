package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"golang.org/x/exp/slices"
)

func main() {
	log.Fatal(serve())
}

var basetmpls = template.Must(template.ParseFS(tmpls, "base.html"))

func serve() error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		basetmpls.ExecuteTemplate(w, "base.html", nil)
	})

	http.HandleFunc("/env", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(os.Environ())
	})

	http.HandleFunc("/fs/", browser)
	http.HandleFunc("/run", run)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	log.Println("listening on port", port)
	return http.ListenAndServe(":"+port, nil)
}

func dump(location string, w io.Writer) error {
	b, err := os.ReadFile(location)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(hex.Dump(b)))
	return err
}

var runtmpls = template.Must(template.ParseFS(tmpls, "run.html", "base.html"))

type RunPipeline struct {
	Error  bool
	Output string
	Code   string
}

func shell(command string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

var space = regexp.MustCompile(`\s+`)

func run(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		runtmpls.ExecuteTemplate(w, "base.html", RunPipeline{})
		return
	}

	if r.Method == "POST" {
		r.ParseForm()
		code := r.Form.Get("code")
		code = space.ReplaceAllString(strings.TrimSpace(code), " ")
		output, errors, err := shell(code)
		if err != nil {
			output += "\n" + err.Error() + "\n"
		}
		if errors != "" {
			output += "\n" + errors + "\n"
		}
		pipeline := RunPipeline{
			Error:  err != nil || errors != "",
			Output: output,
			Code:   code,
		}
		runtmpls.ExecuteTemplate(w, "base.html", pipeline)
	}
}

//go:embed *.html
var tmpls embed.FS

var dirtmpls = template.Must(template.ParseFS(tmpls, "dir.html", "base.html"))

type File struct {
	Name  string
	URL   string
	Size  int64
	IsDir bool
	Mode  string
	Date  string
	Time  string
}

const fs = "/fs"

func directory(location string, w io.Writer) error {
	if location != "/" {
		location = strings.TrimRight(location, "/") + "/"
	}
	entries, err := os.ReadDir(location)
	if err != nil {
		return err
	}
	files := make([]*File, len(entries))
	for i, e := range entries {
		name := e.Name()
		path := location + name
		fi, err := e.Info()
		if err != nil {
			return err
		}
		files[i] = &File{
			Name:  name,
			IsDir: e.IsDir(),
			URL:   fs + path,
			Date:  fi.ModTime().Format("2006-01-02"),
			Time:  fi.ModTime().Format("15:04:05"),
			Mode:  fi.Mode().String(),
			Size:  fi.Size(),
		}
	}
	slices.SortFunc(files, func(a, b *File) bool {
		if a.IsDir != b.IsDir {
			return a.IsDir && !b.IsDir
		}
		return a.Name < b.Name
	})
	var parent string
	if location != "/" {
		parent = fs + filepath.Dir(strings.TrimSuffix(location, "/"))
	}
	pipeline := &struct {
		Files  []*File
		Parent string
	}{files, parent}
	return dirtmpls.ExecuteTemplate(w, "base.html", pipeline)
}

func browser(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, fs), "/")
	if path == "" {
		path = "/"
	}

	path, err := url.QueryUnescape(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fi, err := os.Stat(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if fi.IsDir() {
		if r.URL.Query().Get("tar") != "" {
			err := targz(path, w)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			w.Header().Add("Content-Type", "text/html; charset=utf-8")
			err := directory(path, w)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else {
		if r.URL.Query().Get("view") != "" {
			err := dump(path, w)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.ServeFile(w, r, path)
		}
	}
}

func targz(path string, w http.ResponseWriter) error {
	w.Header().Add("Content-Type", "application/tar+gzip")
	n := filepath.Base(path) + ".tar.gz"
	cd := fmt.Sprintf("attachement; filename=\"%s\"", n)
	w.Header().Add("Content-Disposition", cd)
	return archive(path, w)
}

func archive(root string, w io.Writer) error {
	_, err := os.Stat(root)
	if err != nil {
		return err
	}

	gzw := gzip.NewWriter(w)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	return filepath.Walk(root, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}
		header.Name = strings.TrimPrefix(strings.Replace(file, root, "", -1), string(filepath.Separator))
		err = tw.WriteHeader(header)
		if err != nil {
			return err
		}
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		if err != nil {
			return err
		}
		return nil
	})
}
