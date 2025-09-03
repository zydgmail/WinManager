package guac

import (
	"sync"
)

// SimpleTunnel represents a simple Guacamole tunnel
type SimpleTunnel struct {
	stream *Stream
	mutex  sync.RWMutex
}

// NewSimpleTunnel creates a new simple tunnel
func NewSimpleTunnel(stream *Stream) *SimpleTunnel {
	return &SimpleTunnel{
		stream: stream,
	}
}

// AcquireReader acquires a reader for the tunnel
func (t *SimpleTunnel) AcquireReader() *TunnelReader {
	return &TunnelReader{
		tunnel: t,
	}
}

// AcquireWriter acquires a writer for the tunnel
func (t *SimpleTunnel) AcquireWriter() *TunnelWriter {
	return &TunnelWriter{
		tunnel: t,
	}
}

// Close closes the tunnel
func (t *SimpleTunnel) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	if t.stream != nil {
		return t.stream.Close()
	}
	return nil
}

// TunnelReader provides reading functionality for the tunnel
type TunnelReader struct {
	tunnel *SimpleTunnel
}

// ReadSome reads some data from the tunnel
func (r *TunnelReader) ReadSome() ([]byte, error) {
	r.tunnel.mutex.RLock()
	defer r.tunnel.mutex.RUnlock()
	
	if r.tunnel.stream == nil {
		return nil, ErrConnectionClosed.NewError("隧道已关闭")
	}
	
	return r.tunnel.stream.ReadSome()
}

// Available returns true if data is available
func (r *TunnelReader) Available() bool {
	r.tunnel.mutex.RLock()
	defer r.tunnel.mutex.RUnlock()
	
	if r.tunnel.stream == nil {
		return false
	}
	
	return r.tunnel.stream.Available()
}

// TunnelWriter provides writing functionality for the tunnel
type TunnelWriter struct {
	tunnel *SimpleTunnel
}

// Write writes data to the tunnel
func (w *TunnelWriter) Write(data []byte) (int, error) {
	w.tunnel.mutex.RLock()
	defer w.tunnel.mutex.RUnlock()
	
	if w.tunnel.stream == nil {
		return 0, ErrConnectionClosed.NewError("隧道已关闭")
	}
	
	return w.tunnel.stream.Write(data)
}
