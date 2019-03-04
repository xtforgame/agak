// https://gist.github.com/harlow/dbcd639cf8d396a2ab73
package requestsender

import (
	"net/http"
)

type RequestResult struct {
	Response *http.Response
	Header   http.Header
	Body     []byte
}

type ResponseValidator = func(*RequestResult) error
