package testserver

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
)

// EmptyResponseHandler returns HTTP 204 for any response
// except for the overrides specified via the PathToResponseCode
type EmptyResponseHandler struct {
	PathToResponseCode map[string]int
}

func (s *EmptyResponseHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	responseCode, ok := s.PathToResponseCode[req.URL.Path]
	if ok {
		w.WriteHeader(responseCode)
	} else {
		w.WriteHeader(204)
	}
}

type loggingHandler struct {
	//TODO: add mutexes
	log  bytes.Buffer
	next http.Handler
}

func (s *loggingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(&s.log, "%s %s\n", req.Method, req.URL)
	log.Printf("%s %s\n", req.Method, req.URL)
	s.next.ServeHTTP(w, req)
}

func StartNewTestServer() *TestServer {
	return StartNewTestServerBinding(":0")
}

func StartNewTestServerBinding(binding string) *TestServer {

	testServer := NewTestServer()
	testServer.Binding = binding
	testServer.Start()
	return &testServer
}

func NewTestServer() TestServer {
	return TestServer{Binding: ":0", PathToResponseCode: map[string]int{}}
}

type TestServer struct {
	Binding            string
	PathToResponseCode map[string]int

	server  *http.Server
	addr    *net.TCPAddr
	handler *loggingHandler
}

func (s *TestServer) Start() {
	s.handler = &loggingHandler{
		next: &EmptyResponseHandler{s.PathToResponseCode},
	}

	//TODO: does listener have to be stopped explicitly to free up the port?
	listener, err := net.Listen("tcp", s.Binding)
	if err != nil {
		log.Fatalf("Couldn't start test server on a binding %s, %q", s.Binding, err)
	}
	s.addr = listener.Addr().(*net.TCPAddr)

	s.server = &http.Server{
		Handler: s.handler,
	}
	go s.server.Serve(listener) //TODO: wait till started?
}

func (s TestServer) Addr() string {
	return fmt.Sprintf("http://localhost:%d", s.addr.Port)
}

func (s TestServer) AccessLog() string {
	return s.handler.log.String()
}

func (s *TestServer) Close() {
	s.server.Shutdown(nil)
}
