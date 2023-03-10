package main

import (
	"context"
	"fmt"
	"github.com/kirill0909/balancer/spamer"
	"github.com/spf13/viper"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	LOG_FILE_PATH string = "logs/balancer.log"
	Attempts      int    = iota
	Retry
)

var (
	logger *log.Logger
)

// Server holds the data about a server
type Server struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

// SetAlive for this backend
func (b *Server) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

// IsAlive returns true when backend is alive
func (b *Server) IsAlive() (alive bool) {
	b.mux.RLock()
	alive = b.Alive
	b.mux.RUnlock()
	return
}

// ServerPool holds information about reachable backends
type ServerPool struct {
	backends []*Server
	current  uint64
}

// AddServer to the server pool
func (s *ServerPool) AddServer(server *Server) {
	s.backends = append(s.backends, server)
}

// NextIndex atomically increase the counter and return an index
func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}

// MarkServerStatus changes a status of a backend
func (s *ServerPool) MarkServerStatus(serverUrl *url.URL, alive bool) {
	for _, b := range s.backends {
		if b.URL.String() == serverUrl.String() {
			b.SetAlive(alive)
			break
		}
	}
}

// GetNextPeer returns next active peer to take a connection
func (s *ServerPool) GetNextPeer() *Server {
	next := s.NextIndex()
	l := len(s.backends) + next
	for i := next; i < l; i++ {
		idx := i % len(s.backends)
		if s.backends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.backends[idx]
		}
	}
	return nil
}

// HealthCheck pings the backends and update the status
func (s *ServerPool) HealthCheck(logger *log.Logger) {
	for _, b := range s.backends {
		status := "up"
		alive := isServerAlive(b.URL, logger)
		b.SetAlive(alive)
		if !alive {
			status = "down"
		}
		logger.Printf("%s [%s]\n", b.URL, status)
	}
}

// GetAttemptsFromContext returns the attempts for request
func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(Attempts).(int); ok {
		return attempts
	}
	return 1
}

// GetAttemptsFromContext returns the attempts for request
func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}

// lb load balances the incoming request
func lb(w http.ResponseWriter, r *http.Request) {
	attempts := GetAttemptsFromContext(r)
	if attempts > 3 {
		log.Printf("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}

	peer := serverPool.GetNextPeer()
	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

// isAlive checks whether a server is Alive by establishing a TCP connection
func isServerAlive(u *url.URL, logger *log.Logger) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		logger.Println("Site unreachable, error: ", err)
		return false
	}
	defer conn.Close()
	return true
}

// healthCheck runs a routine for check status of the servers every 20 seconds
func healthCheck(logger *log.Logger) {
	t := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-t.C:
			logger.Println("Starting health check...")
			serverPool.HealthCheck(logger)
			logger.Println("Health check completed")
		}
	}
}

func initConfig() error {
	viper.AddConfigPath("config")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}

var serverPool ServerPool

func main() {

	file, err := os.OpenFile(LOG_FILE_PATH, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		log.Fatalf("error reading log file: %s\n", err.Error())
	}
	defer file.Close()

	logger = log.New(file, "balancer log ", log.LstdFlags)

	if err := initConfig(); err != nil {
		logger.Fatalf("error occured while parsing config file: %s\n", err.Error())
	}

	serverList := viper.GetString("webs")
	port := viper.GetString("port")

	// parse servers
	servers := strings.Split(serverList, ",")
	for _, server := range servers {
		serverUrl, err := url.Parse(server)
		if err != nil {
			logger.Fatal(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(serverUrl)
		proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
			logger.Printf("[%s] %s\n", serverUrl.Host, e.Error())
			retries := GetRetryFromContext(request)
			if retries < 3 {
				select {
				case <-time.After(100 * time.Second):
					ctx := context.WithValue(request.Context(), Retry, retries+1)
					proxy.ServeHTTP(writer, request.WithContext(ctx))
				}
				return
			}

			// after 3 retries, mark this backend as down
			serverPool.MarkServerStatus(serverUrl, false)

			// if the same request routing for few attempts with different backends, increase the count
			attempts := GetAttemptsFromContext(request)
			logger.Printf("%s(%s) Attempting retry %d\n", request.RemoteAddr, request.URL.Path, attempts)
			ctx := context.WithValue(request.Context(), Attempts, attempts+1)
			lb(writer, request.WithContext(ctx))
		}

		serverPool.AddServer(&Server{
			URL:          serverUrl,
			Alive:        true,
			ReverseProxy: proxy,
		})
		logger.Printf("Configured server: %s\n", serverUrl)
	}

	// create http server
	server := http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: http.HandlerFunc(lb),
	}

	// start health checking
	go healthCheck(logger)
	// run spamer
	go spamer.DoSpam()

	logger.Printf("Load Balancer started at :%v\n", port)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			logger.Fatal(err)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGKILL, os.Interrupt)
	<-quit

}
