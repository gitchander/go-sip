package sipnet

import (
	"io"
	"strconv"
)

// SIP request methods.
const (
	MethodInvite   = "INVITE"
	MethodAck      = "ACK"
	MethodBye      = "BYE"
	MethodCancel   = "CANCEL"
	MethodRegister = "REGISTER"
	MethodOptions  = "OPTIONS"
	MethodInfo     = "INFO"
)

// Request represents a SIP request (i.e. a message sent by a UAC to a UAS).
type Request struct {
	Method string
	Server string
	Proto  string

	Header Header

	Body []byte
}

// NewRequest returns a new request.
func NewRequest() *Request {
	return &Request{
		Proto:  ProtoSIP,
		Header: make(Header),
	}
}

var _ io.WriterTo = &Request{}

func (r *Request) WriteTo(w io.Writer) (n int64, err error) {
	ni, err := w.Write([]byte(r.Method + " " + r.Server + " " + r.Proto + "\r\n"))
	if err != nil {
		return int64(ni), err
	}

	r.Header.Set("Content-Length", strconv.Itoa(len(r.Body)))

	_, err = r.Header.WriteTo(w)
	if err != nil {
		return 0, err
	}

	ni, err = w.Write(r.Body)
	return int64(ni), err
}

// WriteTo writes the request data to a Conn. It automatically adds a
// a Content-Length to the header, calls Flush() on the Conn.
func (r *Request) WriteToConn(conn *Conn) error {
	_, err := r.WriteTo(conn)
	if err != nil {
		return err
	}
	return conn.Flush()
}
