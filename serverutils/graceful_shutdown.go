package serverutils

import (
	"fmt"
	"context"
	"github.com/xtforgame/agak/utils"
	"net"
	"net/http"
	"time"
)

func RunAndWaitGracefulShutdown(server *http.Server) {
	chBack := make(chan error)
	go func() {
		_ = server.ListenAndServe()
		fmt.Println("server shutdown")
		time.Sleep(time.Second * 1)
		fmt.Println("server shutdown delay ended")
		chBack <- nil
	}()

	utils.HandleSIGINTandSIGTERM()

	// Stop the service gracefully.
	fmt.Println("start shutdown")
	fmt.Println("shutdown result :", server.Shutdown(context.Background()))
	fmt.Println("shutdown called")

	fmt.Println("error :", <-chBack)
	fmt.Println("exit")
}

func ListenAndServeUnixSoclet(srv *http.Server, filepath string) error {
	ln, err := net.Listen("unix", filepath)
	if err != nil {
		fmt.Println("err :", err)
		return err
	}
	return srv.Serve(ln)
}

func RunAndWaitGracefulShutdownUnixSoclet(server *http.Server, filepath string) {
	chBack := make(chan error)
	go func() {
		_ = ListenAndServeUnixSoclet(server, filepath)
		fmt.Println("server shutdown")
		time.Sleep(time.Second * 1)
		fmt.Println("server shutdown delay ended")
		chBack <- nil
	}()

	utils.HandleSIGINTandSIGTERM()

	// Stop the service gracefully.
	fmt.Println("start shutdown")
	fmt.Println("shutdown result :", server.Shutdown(context.Background()))
	fmt.Println("shutdown called")

	fmt.Println("error :", <-chBack)
	fmt.Println("exit")
}
