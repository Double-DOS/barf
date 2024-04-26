package websocket

import (
	"bufio"
	"net"
)

// BufferPool represents a pool of buffers. The *sync.Pool type satisfies this
// interface.  The type of the value stored in a pool is not specified.
type BufferPool interface {
	// Get gets a value from the pool or returns nil if the pool is empty.
	Get() interface{}
	// Put adds a value to the pool.
	Put(interface{})
}

// Storm represents the new Websocket Connection created by the StormBreaker;
// holds the connection information from the original http.Connection
type Storm struct {
	// instance of the hijacked raw http connection
	RawConn net.Conn

	// writing fields
	bufW []byte
	mu   chan struct{} // mutex lock for writing
}

func NewStorm(rawConn net.Conn, bufR *bufio.Reader, bufW []byte, writeBufferPool BufferPool, readBufferSize, writeBufferSize int) *Storm {
	if bufR == nil {
		if readBufferSize == 0 {
			readBufferSize = defaultReadBufferSize
		} else if readBufferSize < maxControlFramePayloadSize {
			// must be large enough for control frame
			readBufferSize = maxControlFramePayloadSize
		}
		bufR = bufio.NewReaderSize(rawConn, readBufferSize)

	}
	if writeBufferSize <= 0 {
		writeBufferSize = defaultWriteBufferSize
	}
	writeBufferSize += maxFrameHeaderSize

	if bufW == nil && writeBufferPool == nil {
		bufW = make([]byte, writeBufferSize)
	}

	mu := make(chan struct{}, 1)
	mu <- struct{}{}
	storm := &Storm{
		RawConn: rawConn,
		mu:      mu,
	}

	return storm
}
