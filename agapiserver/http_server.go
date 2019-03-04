// https://github.com/gorilla/websocket/blob/master/examples/echo/server.go
// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package agapiserver

import (
	// "bytes"
	// "encoding/json"
	// "fmt"
	"github.com/go-chi/chi"
	// funk "github.com/thoas/go-funk"
	"github.com/xtforgame/agak/gbcore"
	"github.com/xtforgame/agak/serverutils"
	"net/http"
	// "sort"
	"encoding/json"
	"github.com/xtforgame/cmdraida/crbasic"
	"github.com/xtforgame/cmdraida/crcore"
	// "github.com/xtforgame/cmdraida/t1"
	"os"
	// "strings"
)

type HttpServer struct {
	server      *http.Server
	router      *chi.Mux
	taskManager *crbasic.TaskManagerBase
}

func NewHttpServer() *HttpServer {
	r := chi.NewRouter()
	return &HttpServer{
		server: &http.Server{
			Addr:    ":8080",
			Handler: r,
		},
		router: r,
	}
}

var runtimeFolder = "./runtime"

func (hs *HttpServer) Init() {
	os.RemoveAll(runtimeFolder)
	os.MkdirAll(runtimeFolder, os.ModePerm)
	hs.taskManager = crbasic.NewTaskManager(runtimeFolder, gbcore.NewReporterT1)
	hs.taskManager.Init()

	// hs.router.FileServer("/", http.Dir("web/"))
	// FileServer(hs.router, "/assets", http.Dir("./assets"))
	hs.router.HandleFunc("/echo", TestHandleWebsocket)
	hs.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		task := hs.taskManager.RunTask(crcore.CommandType{
			Command: "sh",
			Args:    []string{"-c", "echo xxx;sleep 2;echo ooo"},
			Timeouts: crcore.TimeoutsType{
				Proccess:    1000,
				AfterKilled: 1500,
			},
		})
		if jsonBytes, err := json.Marshal(task.ResultLog()); err == nil {
			w.Write(jsonBytes)
			return
		}
		w.Write([]byte("[]"))
	})

	hs.router.Get("/test1", func(w http.ResponseWriter, r *http.Request) {
		task := hs.taskManager.RunTask(crcore.CommandType{
			Command: "bash",
			Args:    []string{"-c", "echo xxx;sleep 2;echo ooo"},
			Timeouts: crcore.TimeoutsType{
				Proccess:    1000,
				AfterKilled: 1500,
			},
		})
		if jsonBytes, err := json.Marshal(task.ResultLog()); err == nil {
			w.Write(jsonBytes)
			return
		}
		w.Write([]byte("[]"))
	})

	hs.router.Get("/test2", func(w http.ResponseWriter, r *http.Request) {
		task := hs.taskManager.RunTask(crcore.CommandType{
			Command: "bash",
			Args:    []string{"-c", "echo $XXX;go version;sleep 2;echo ooo"},
			Timeouts: crcore.TimeoutsType{
				Proccess:    1000,
				AfterKilled: 1500,
			},
			Env: []string{"XXX=1"},
			Dir: "/",
		})
		if jsonBytes, err := json.Marshal(task.ResultLog()); err == nil {
			w.Write(jsonBytes)
			return
		}
		w.Write([]byte("[]"))
	})
}

func (hs *HttpServer) Start() {
	serverutils.RunAndWaitGracefulShutdown(hs.server)
}
