package websocket

import (
	"bufio"
	"errors"
	"net/http"

	logger "github.com/opensaucerer/barf/log"
)

// StormBreaker represents the configuration for the websocket connection.
// It collects parameters to upgrade the http connection into a new websocket.
type StormBreaker struct {
	WriteBufferSize int
	ReadBufferSize  int

	// collecting the function to be called when testing for the valid origin
	// returns boolean indicating whether the origin is valid or not.
	CheckOrigin func(r *http.Request) bool

	WriteBufferPool BufferPool
}

func (sb *StormBreaker) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*Storm, error) {
	// default starting message for a bad handshake error response
	const badHandshake = "websocket: the client is not using the websocket protocol: "

	// we'll check that this connection is really an upgrade connection
	// and uses the right wet of Sec-Websocket-Protocol and other headers that properly
	// describes an http upgrade connection like; Connection Upgrade, Upgrade Webockets; etc.
	// according to RFC 6455
	if !headerListContainsToken(r.Header, "Connection", "upgrade") {
		// todo: return error response
		logger.Error(badHandshake + "websocket token: \"upgrade\" not found in \"Connection\" header")
	}

	if !headerListContainsToken(r.Header, "Upgrade", "websocket") {
		// todo: return error response
		logger.Error(badHandshake + "websocket token: \"websocket\" not found in \"Upgrade\" header")
	}

	if r.Method != http.MethodGet {
		// todo: return error response
		logger.Error(badHandshake + "request method is not GET")
	}

	if !headerListContainsToken(r.Header, "Sec-Websocket-Version", "13") {
		// todo: return error response
		logger.Error("websocket: unsupported version: 13 not found in 'Sec-Websocket-Version' header")
	}

	if _, ok := responseHeader["Sec-Websocket-Extensions"]; ok {
		// todo: return error response
		logger.Error("websocket: application specific 'Sec-WebSocket-Extensions' headers are unsupported")
	}

	// following RFC 6455, we need to ensure that we verify that the origin of this request is allowed
	// this is handled by the application by passing to storm breaker the function to check the origin of the request
	checkOrigin := sb.CheckOrigin
	if checkOrigin == nil {
		checkOrigin = defaultCheckSameOrigin
	}
	if checkOrigin(r) {
		// todo: return error response
		logger.Error("websocket: request origin blocked by StormBreaker.CheckOrigin()")
	}

	challengeKey := r.Header.Get("Sec-Websocket-Key")
	if !isValidChallengeKey(challengeKey) {
		// todo: return error response
		logger.Error("websocket: not a websocket handshake: 'Sec-WebSocket-Key' header must be Base64 encoded value of 16-byte in length")
	}

	h, ok := w.(http.Hijacker)
	if !ok {
		// actually this should return an error in this case
		// we study barf's error handling to get the error out proper
		// todo: return error response
		logger.Error("websocket: error, response writer is not a http.Hijacker")

	}

	var bufRW *bufio.ReadWriter
	rawConn, bufRW, err := h.Hijack()
	if err != nil {
		// todo: return error response
		logger.Error("websocket: error, can not hijack http connection")
	}

	// check that messages are not being sent before the websocket handshake is complete
	if bufRW.Reader.Buffered() > 0 {
		return nil, errors.New("websocket: messages sent before handshake is complete")
	}
	var bufR *bufio.Reader

	if sb.ReadBufferSize == 0 && bufioReaderSize(rawConn, bufRW.Reader) < 256 {
		// use hijacked buffered reader as connection reader. this is an optimization technique
		// it aonly uses the hijacked reader if it is large enough to be used by smaller buffers
		bufR = bufRW.Reader
	}

	buf := bufioWriterBuffer(rawConn, bufRW.Writer)

	var bufW []byte
	if sb.WriteBufferPool == nil && sb.WriteBufferSize == 0 && len(buf) >= maxFrameHeaderSize+256 {
		// Reuse hijacked write buffer as connection buffer. only when no specific Writer Buffer Size is not set.
		bufW = buf
	}

	// todo: proceed with the new stormConnection.
	stormConn := NewStorm(rawConn, bufR, bufW, sb.WriteBufferPool, sb.ReadBufferSize, sb.WriteBufferSize)
	// todo: build the response header in bytes array.

	return stormConn, nil

}
