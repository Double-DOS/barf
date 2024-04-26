package websocket

const (
	maxFrameHeaderSize         = 2 + 8 + 4 // Fixed header + length + mask
	maxControlFramePayloadSize = 125

	// default buffer sizes
	defaultReadBufferSize  = 4096
	defaultWriteBufferSize = 4096
)
