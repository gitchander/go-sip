package sipnet

import (
	"fmt"
	"io"
	"strconv"
)

// Response represents a SIP response (i.e. a message sent by a UAS to a UAC).
type Response struct {
	Request *Request

	StatusCode int
	Status     string
	Proto      string
	Header     Header
	Body       []byte
}

// NewResponse returns a new response.
func NewResponse(Request *Request) *Response {
	return &Response{
		Request: Request,
		Proto:   ProtoSIP,
		Header:  make(Header),
	}
}

var _ io.WriterTo = &Response{}

// WriteTo writes the response data to a Conn. It automatically adds a
// a Content-Length, CSeq, Call-ID and Via header. It also sets the Status message
// appropriately and automatically calls Flush() on the Conn.
func (r *Response) WriteTo(w io.Writer) (n int64, err error) {

	ni, err := fmt.Fprintf(w, "%s %d %s\r\n", r.Proto, r.StatusCode, StatusText(r.StatusCode))
	if err != nil {
		return int64(ni), err
	}

	// ni, err := w.Write([]byte(r.Proto + " " +
	// 	strconv.Itoa(r.StatusCode) + " " +
	// 	StatusText(r.StatusCode) +
	// 	"\r\n"))
	// if err != nil {
	// 	return int64(ni), err
	// }

	r.Header.Set("Content-Length", strconv.Itoa(len(r.Body)))

	//	if r.Request != nil {

	//		// Via
	//		{
	//			requestVia, err := ParseVia(r.Request.Header.Get("Via"))
	//			if err != nil {
	//				return 0, err
	//			}

	//			// ipPort := strings.Split(conn.Addr().String(), ":")
	//			// requestVia.Arguments.Set("received", ipPort[0])
	//			// requestVia.Arguments.Set("rport", ipPort[1])

	//			r.Header.Set("Via", requestVia.String())
	//		}

	//		r.Header.Set("CSeq", r.Request.Header.Get("CSeq"))
	//		r.Header.Set("Call-ID", r.Request.Header.Get("Call-ID"))
	//	}

	n, err = r.Header.WriteTo(w)
	if err != nil {
		return n, err
	}

	ni, err = w.Write(r.Body)
	return int64(ni), err
}

func (r *Response) WriteToConn(conn *Conn) error {
	_, err := r.WriteTo(conn)
	if err != nil {
		return err
	}
	return conn.Flush()
}

// BadRequest responds to a Conn with a StatusBadRequest for convenience.
func (r *Response) BadRequest(conn *Conn, reason string) {
	r.StatusCode = StatusBadRequest
	r.Header.Set("Reason-Phrase", reason)
	r.WriteTo(conn)
}

// ServerError responds to a Conn with a StatusServerInternalError
// for convenience.
func (r *Response) ServerError(conn *Conn, reason string) {
	r.StatusCode = StatusServerInternalError
	r.Header.Set("Reason-Phrase", reason)
	r.WriteTo(conn)
}
