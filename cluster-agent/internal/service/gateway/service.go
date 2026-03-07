package gateway

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/edgebase/cluster-agent/internal/model"
	"github.com/google/uuid"
)

type RouteFetcher interface {
	FetchGatewayRoutes(ctx context.Context, clusterID uuid.UUID) ([]model.GatewayRoute, error)
}

type Service struct {
	fetcher    RouteFetcher
	clusterID  uuid.UUID
	refresh    time.Duration
	listenAddr string
	logger     *slog.Logger

	mu     sync.RWMutex
	routes []model.GatewayRoute
}

func New(fetcher RouteFetcher, clusterID uuid.UUID, refresh time.Duration, listenAddr string, logger *slog.Logger) *Service {
	return &Service{
		fetcher:    fetcher,
		clusterID:  clusterID,
		refresh:    refresh,
		listenAddr: listenAddr,
		logger:     logger,
		routes:     []model.GatewayRoute{},
	}
}

func (s *Service) Start(ctx context.Context) error {
	if err := s.refreshRoutes(ctx); err != nil {
		s.logger.Warn("initial gateway route refresh failed", "error", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)

	server := &http.Server{
		Addr:    s.listenAddr,
		Handler: mux,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	go s.refreshLoop(ctx)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	s.logger.Info("gateway listening", "addr", s.listenAddr)
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Service) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(s.refresh)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.refreshRoutes(ctx); err != nil {
				s.logger.Warn("gateway route refresh failed", "error", err)
			}
		}
	}
}

func (s *Service) refreshRoutes(ctx context.Context) error {
	routes, err := s.fetcher.FetchGatewayRoutes(ctx, s.clusterID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.routes = routes
	s.mu.Unlock()
	return nil
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	route, ok := s.matchRoute(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	targetURL := &url.URL{
		Scheme: "http",
		Host:   route.ServiceName + "." + route.Namespace + ".svc.cluster.local",
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		http.Error(rw, "gateway upstream error", http.StatusBadGateway)
	}
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
		req.URL.Path = "/invoke"
		req.Header.Set("X-EdgeBase-Route-ID", route.ID.String())
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Forwarded-Method", r.Method)
	}
	if route.TimeoutMs > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(route.TimeoutMs)*time.Millisecond)
		defer cancel()
		r = r.WithContext(ctx)
	}
	proxy.ServeHTTP(w, r)
}

func (s *Service) matchRoute(r *http.Request) (model.GatewayRoute, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	host := stripPort(r.Host)
	path := r.URL.Path
	method := strings.ToUpper(r.Method)

	for _, route := range s.routes {
		if route.Host != "" && route.Host != host {
			continue
		}
		if route.Path != path {
			continue
		}
		for _, candidate := range route.Methods {
			if strings.EqualFold(candidate, method) {
				return route, true
			}
		}
	}
	return model.GatewayRoute{}, false
}

func stripPort(host string) string {
	if idx := strings.Index(host, ":"); idx >= 0 {
		return host[:idx]
	}
	return host
}

