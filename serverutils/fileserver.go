// https://github.com/gorilla/websocket/blob/master/examples/echo/server.go
// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package serverutils

import (
	// "bytes"
	// "encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	// funk "github.com/thoas/go-funk"
	// "github.com/xtforgame/agak/gbcore"
	"net/http"
	// // "sort"
	// "encoding/json"
	// "github.com/xtforgame/cmdraida/crbasic"
	// "github.com/xtforgame/cmdraida/crcore"
	// // "github.com/xtforgame/cmdraida/t1"
	// "os"
	"strings"
)

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	fs := http.StripPrefix(path, http.FileServer(root))
	fmt.Println("path :", path)
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))
}
