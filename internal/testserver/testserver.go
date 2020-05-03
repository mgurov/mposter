package testserver

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
)

type testResponseHandler struct {
	PathToHandler map[string]func(http.ResponseWriter, *http.Request)
}

func (s *testResponseHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	handler, ok := s.PathToHandler[req.URL.Path]
	if ok {
		handler(w, req)
	} else {
		w.WriteHeader(204)
	}
}

func MakeEmptyResponseHandler(httpStatus int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(httpStatus)
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
	return TestServer{Binding: ":0", PathToHandler: map[string]func(http.ResponseWriter, *http.Request){}}
}

type TestServer struct {
	Binding       string
	PathToHandler map[string]func(http.ResponseWriter, *http.Request)

	server  *http.Server
	addr    *net.TCPAddr
	handler *loggingHandler
}

func (s *TestServer) Start() {
	s.handler = &loggingHandler{
		next: &testResponseHandler{s.PathToHandler},
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

func (s *TestServer) Shutdown() {
	s.server.Shutdown(context.Background())
}

func (s *TestServer) ReturnEmptyResponseWithHttpStatus(path string, responseCode int) {
	s.PathToHandler[path] = MakeEmptyResponseHandler(responseCode)
}

func (s *TestServer) RegisterHandler(path string, handler func(http.ResponseWriter, *http.Request)) {
	s.PathToHandler[path] = handler
}
