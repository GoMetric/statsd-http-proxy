package proxy

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/GoMetric/go-statsd-client"
	"github.com/GoMetric/statsd-http-proxy/proxy/routehandler"
	"github.com/GoMetric/statsd-http-proxy/proxy/router"
)

// Server is a proxy server between HTTP REST API and UDP Connection to StatsD
type Server struct {
	httpServer   *http.Server
	statsdClient *statsd.Client
	tlsCert      string
	tlsKey       string
}

// NewServer creates new instance of StatsD HTTP Proxy
func NewServer(
	httpHost string,
	httpPort int,
	httpReadTimeout int,
	httpWriteTimeout int,
	httpIdleTimeout int,
	statsdHost string,
	statsdPort int,
	tlsCert string,
	tlsKey string,
	metricPrefix string,
	tokenSecret string,
	verbose bool,
	httpRouterName string,
) *Server {
	// configure logging
	var logOutput io.Writer
	if verbose == true {
		logOutput = os.Stderr
	} else {
		logOutput = ioutil.Discard
	}

	log.SetOutput(logOutput)

	logger := log.New(logOutput, "", log.LstdFlags)

	// create StatsD Client
	statsdClient := statsd.NewClient(statsdHost, statsdPort)

	// build route handler
	routeHandler := routehandler.NewRouteHandler(
		statsdClient,
		metricPrefix,
	)

	// build router
	var httpServerHandler http.Handler
	switch httpRouterName {
	case "GorillaMux":
		httpServerHandler = router.NewGorillaMuxRouter(routeHandler, tokenSecret)
	case "HttpRouter":
		httpServerHandler = router.NewHTTPRouter(routeHandler, tokenSecret)
	default:
		panic("Passed HTTP router not supported")
	}

	// get HTTP server address to bind
	httpAddress := fmt.Sprintf("%s:%d", httpHost, httpPort)
	log.Printf("Starting HTTP server at %s", httpAddress)

	// create http server
	httpServer := &http.Server{
		Addr:           httpAddress,
		Handler:        httpServerHandler,
		ErrorLog:       logger,
		ReadTimeout:    time.Duration(httpReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(httpWriteTimeout) * time.Second,
		IdleTimeout:    time.Duration(httpIdleTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	statsdHTTPProxyServer := Server{
		httpServer,
		statsdClient,
		tlsCert,
		tlsKey,
	}

	return &statsdHTTPProxyServer
}

// Listen starts listening HTTP connections
func (proxyServer *Server) Listen() {
	// open StatsD connection
	proxyServer.statsdClient.Open()
	defer proxyServer.statsdClient.Close()

	// start HTTP/HTTPS server
	var err error
	if len(proxyServer.tlsCert) > 0 && len(proxyServer.tlsKey) > 0 {
		err = proxyServer.httpServer.ListenAndServeTLS(proxyServer.tlsCert, proxyServer.tlsKey)
	} else {
		err = proxyServer.httpServer.ListenAndServe()
	}

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
