package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"syscall"
)

//SetSocketOptions functions sets IP_TRANSPARENT flag on given socket (c syscall.RawConn)
func SetSocketOptions(network string, address string, c syscall.RawConn) error {

	var fn = func(s uintptr) {
		var setErr error
		var getErr error
		setErr = syscall.SetsockoptInt(int(s), syscall.SOL_IP, syscall.IP_TRANSPARENT, 1)
		if setErr != nil {
			log.Fatal(setErr)
		}

		val, getErr := syscall.GetsockoptInt(int(s), syscall.SOL_IP, syscall.IP_TRANSPARENT)
		if getErr != nil {
			log.Fatal(getErr)
		}
		log.Printf("value of IP_TRANSPARENT option is: %d", int(val))
	}
	if err := c.Control(fn); err != nil {
		return err
	}

	return nil

}

func main() {

	http.HandleFunc("/", TransparentHttpProxy)

	// here we are creating custom listener with transparent socket, possible with Go 1.11+
	lc := net.ListenConfig{Control: SetSocketOptions}
	listener, _ := lc.Listen(context.Background(), "tcp", ":8888")

	log.Printf("Starting http proxy")
	log.Fatal(http.Serve(listener, nil))

}

func TransparentHttpProxy(w http.ResponseWriter, r *http.Request) {

	director := func(target *http.Request) {
		target.URL.Scheme = "http"
		target.URL.Path = r.URL.Path
		target.Header.Set("Pass-Via-Go-Proxy", "1")
		/*
			Line below of this comment this is the quite tricky part of the configuration,
			necessary to make transparent proxy working.

			From http.LocalAddrContextKey we can get address:port destination of client requst.
			In fact address:port values from http.LocalAddrContextKey,
			are the values from socket dynamicly created by tproxy.
			This will be used to create a connection between the proxy and the destination,
			to which the client request will be pass.
		*/
		target.URL.Host = fmt.Sprint(r.Context().Value(http.LocalAddrContextKey))
	}
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(w, r)

}
