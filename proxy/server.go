package proxy

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"github.com/GoMetric/statsd-http-proxy/proxy/routehandler"
	"github.com/GoMetric/go-statsd-client"
)

// Server is a proxy server between HTTP REST API and UDP Connection to StatsD
type Server struct {
	httpServer *http.Server
	statsdClient *statsd.Client
	tlsCert string
	tlsKey  string
}

// NewServer creates new instance of StatsD HTTP Proxy
func NewServer(
	httpHost string,
	httpPort int,
	statsdHost string,
	statsdPort int,
	tlsCert string,
	tlsKey string,
	metricPrefix string,
	tokenSecret string,
) *Server {
	// create StatsD Client
	statsdClient := statsd.NewClient(statsdHost, statsdPort)

	// build router
	router := routehandler.NewRouter(
		statsdClient,
		metricPrefix,
		tokenSecret,
	)

	// get HTTP server address to bind
	httpAddress := fmt.Sprintf("%s:%d", httpHost, httpPort)
	log.Printf("Starting HTTP server at %s", httpAddress)

	// create http server
	httpServer := &http.Server{
		Addr:           httpAddress,
		Handler:        router,
		ReadTimeout:    1 * time.Second,
		WriteTimeout:   1 * time.Second,
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
