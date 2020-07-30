package go_usbmuxd_device

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// HTTPProxy is our main struct for proxy releated attributes and methods
type HTTPProxy struct {
	// The transport used to send proxy requests to actual server.
	// If nil, http.DefaultTransport is used.
	Transport  http.RoundTripper
	Credential string
}

// NewProxy returns a new HTTPProxy object
func newHTTPProxy() *HTTPProxy {
	return &HTTPProxy{}
}

func parseHostPort(addr string) (host string, port int) {
	fields := strings.SplitN(addr, ":", 2)
	if len(fields) == 1 {
		return addr, 80
	}
	port, _ = strconv.Atoi(fields[1])
	return fields[0], port
}

// NewUSBHTTPProxy used to proxy connection to iPhone
func NewUSBHTTPProxy(usbhub *USBHub) (*HTTPProxy, error) {
	if usbhub == nil {
		usbhub = NewUSBHub()
	}
	devCh, err := usbhub.deviceListenAttached()
	if err != nil {
		return nil, err
	}

	devIDs := make(map[string]int)
	go func() {
		for dev := range devCh {
			log.Println("Device:", dev)
			devIDs[dev.SerialNumber] = dev.DeviceID
		}
	}()

	// network always http
	// addr is device UDID:PORT
	dialDevice := func(network, addr string) (net.Conn, error) {
		udid, port := parseHostPort(addr)
		devID, ok := devIDs[udid]
		if !ok {
			return nil, fmt.Errorf("Device %s not connected", udid)
		}
		return usbhub.CreateConnect(devID, port)
	}

	return &HTTPProxy{
		Transport: &http.Transport{
			Dial: dialDevice,
		},
	}, nil
}

func (p *HTTPProxy) handleConnect(rw http.ResponseWriter, req *http.Request) {
	host := req.URL.Host

	hij, ok := rw.(http.Hijacker)
	if !ok {
		panic("HTTP Server does not support hijacking")
	}

	client, _, err := hij.Hijack()
	if err != nil {
		return
	}
	defer client.Close()

	client.Write([]byte("HTTP/1.0 200 Connection Established\r\n\r\n"))
	server, err := net.Dial("tcp", host)
	if err != nil {
		return
	}
	defer server.Close()

	go io.Copy(server, client)
	io.Copy(client, server)
}

// https://zh.wikipedia.org/wiki/HTTP%E5%9F%BA%E6%9C%AC%E8%AE%A4%E8%AF%81
func proxyBasicAuth(r *http.Request) (username, password string, ok bool) {
	auth := r.Header.Get("Proxy-Authorization")
	if auth == "" {
		return
	}
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

func (p *HTTPProxy) proxyAuthCheck(r *http.Request) (ok bool) {
	if p.Credential == "" { // no auth
		return true
	}
	auth := r.Header.Get("Proxy-Authorization")
	if auth == "" {
		return
	}
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return
	}
	credential := auth[len(prefix):]
	return credential == p.Credential
}

func (p *HTTPProxy) handleProxyAuth(w http.ResponseWriter, r *http.Request) bool {
	if p.proxyAuthCheck(r) {
		return true
	}
	w.Header().Add("Proxy-Authenticate", "Basic realm=\"*\"")
	w.WriteHeader(http.StatusProxyAuthRequired)
	w.Write(nil)
	return false
}

// ServeHTTP is the main handler for all requests.
func (p *HTTPProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if !p.handleProxyAuth(rw, req) {
		return
	}
	fmt.Printf("Received request %s %s %s\n",
		req.Method,
		req.Host,
		req.RemoteAddr,
	)

	if req.Method == "CONNECT" {
		p.handleConnect(rw, req)
		return
	}

	transport := p.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	// copy the origin request, and modify according to proxy
	// standard and user rules.
	outReq := new(http.Request)
	*outReq = *req // this only does shallow copies of maps

	// Set `x-Forwarded-For` header.
	// `X-Forwarded-For` contains a list of servers delimited by comma and space
	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		if prior, ok := outReq.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		outReq.Header.Set("X-Forwarded-For", clientIP)
	}

	// send the modified request and get response
	res, err := transport.RoundTrip(outReq)
	if err != nil {
		rw.WriteHeader(http.StatusBadGateway)
		rw.Write([]byte(err.Error()))
		return
	}

	// write response back to client, including status code, header and body

	for key, value := range res.Header {
		// Some header item can contains many values
		for _, v := range value {
			rw.Header().Add(key, v)
		}
	}

	rw.WriteHeader(res.StatusCode)
	io.Copy(rw, res.Body)
	res.Body.Close()
}

// func main() {
// 	addr := flag.String("addr", ":8080", "listen address")
// 	auth := flag.String("auth", "", "http auth, eg: susan:hello-kitty")
// 	flag.Parse()

// 	proxy := NewProxy()
// 	if *auth != "" {
// 		proxy.Credential = base64.StdEncoding.EncodeToString([]byte(*auth))
// 	}
// 	fmt.Printf("listening on %s\n", *addr)
// 	http.ListenAndServe(*addr, proxy)
// }
