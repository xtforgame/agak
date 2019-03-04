package serverutils

import (
	"fmt"
	"github.com/xtforgame/agak/utils"
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
	fmt.Println("shutdown result :", server.Shutdown(nil))
	fmt.Println("shutdown called")

	fmt.Println("error :", <-chBack)
	fmt.Println("exit")
}
