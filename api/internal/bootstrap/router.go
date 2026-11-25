package bootstrap

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Lmare/lightning-playground/api/internal/infra/http/web"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type route struct {
	path     string
	handlers map[string]func(http.ResponseWriter, *http.Request)
}

type router struct {
	routes     map[string]*route
	prefixes   map[string]*route
	metrics    *prometheus.CounterVec
	registry   *prometheus.Registry
	instrument bool
}

// metrics is shared by the app and ops routers: one registry, one counter.
type metrics struct {
	counter  *prometheus.CounterVec
	registry *prometheus.Registry
}

func newMetrics() *metrics {
	reg := prometheus.NewRegistry()
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"handler", "method"},
	)
	reg.MustRegister(requestCounter)
	return &metrics{counter: requestCounter, registry: reg}
}

// newRouter builds a router; only an instrumented router counts its requests.
func newRouter(m *metrics, instrument bool) *router {
	return &router{
		routes:     make(map[string]*route),
		prefixes:   make(map[string]*route),
		metrics:    m.counter,
		registry:   m.registry,
		instrument: instrument,
	}
}

// Routers groups the two listeners of the API: the application (pages, fragments, JSON,
// SSE, assets) and the ops one (probes and metrics, cluster-internal).
type Routers struct {
	App http.Handler
	Ops http.Handler
}

func InitRouters(handlers *handlers) Routers {
	m := newMetrics()
	return Routers{App: initAppRouter(handlers, m), Ops: initOpsRouter(handlers, m)}
}

func initOpsRouter(handlers *handlers, m *metrics) http.Handler {
	router := newRouter(m, false)
	router.add("/health", http.MethodGet, handlers.probe.HandleHealth)
	router.add("/ready", http.MethodGet, handlers.probe.HandleReady)
	router.add("/metrics", http.MethodGet, router.handleMetrics)
	return router
}

func initAppRouter(handlers *handlers, m *metrics) http.Handler {
	router := newRouter(m, true)

	router.add("/", http.MethodGet, handlers.web.HandleHome)
	router.add("/hello", http.MethodGet, handlers.web.HandleHome)
	router.add("/person", http.MethodGet, handlers.web.HandlePersonPage)
	router.add("/node", http.MethodGet, handlers.web.HandleNodePage)
	router.add("/node/panel", http.MethodGet, handlers.web.HandleNodePanel)
	router.addPrefix("/asset/", http.MethodGet, web.StaticHandler().ServeHTTP)

	router.addAPI("/lightning/alias", http.MethodPut, handlers.lightning.HandleUpdateNodeAlias)
	router.addAPI("/lightning/channel", http.MethodPost, handlers.lightning.HandleOpenChannel)
	router.addAPI("/lightning/channel/close", http.MethodPost, handlers.lightning.HandleCloseChannel)
	router.addAPI("/lightning/balance", http.MethodGet, handlers.lightning.HandleGetBalance)
	router.addAPI("/lightning/channels", http.MethodGet, handlers.lightning.HandleListChannels)
	router.addAPI("/lightning/invoice", http.MethodPost, handlers.lightning.HandleCreateInvoice)
	router.addAPI("/lightning/nodes", http.MethodGet, handlers.lightning.HandleListOfNodes)
	router.addAPI("/lightning/nodeInfo", http.MethodGet, handlers.lightning.HandleNodeInfo)
	router.addAPI("/lightning/payment", http.MethodPost, handlers.lightning.HandleMakePayment)
	router.addAPI("/lightning/peer", http.MethodPost, handlers.lightning.HandleAddPeer)
	router.addAPI("/lightning/wallet/unlock", http.MethodPost, handlers.lightning.HandleUnlockWallet)
	router.addAPI("/lightning/uri", http.MethodGet, handlers.lightning.HandleShowUri)

	router.addAPI("/users", http.MethodGet, handlers.user.HandleListUsers)

	router.addAPI("/stream-event", http.MethodGet, handlers.streamEvent.HandleStreamEvent)
	router.addAPI("/version", http.MethodGet, handlers.version.HandleVersion)

	return router
}

// apiPrefix is the versioned prefix of every non-page route (fragments, JSON, SSE).
const apiPrefix = "/api/v1"

// addAPI registers a route under apiPrefix; the metrics label is the prefixed path.
func (router *router) addAPI(path string, verbe string, callback func(http.ResponseWriter, *http.Request)) {
	router.add(apiPrefix+path, verbe, callback)
}

func (router *router) add(path string, verbe string, callback func(http.ResponseWriter, *http.Request)) {
	if _, exist := router.routes[path]; !exist {
		router.routes[path] = &route{path: path, handlers: make(map[string]func(http.ResponseWriter, *http.Request))}
	}
	if router.instrument {
		router.routes[path].handlers[verbe] = router.wrapHandlerWithMetrics(path, callback)
	} else {
		router.routes[path].handlers[verbe] = callback
	}
}

// addPrefix registers a handler for every path under prefix; the metrics label is the prefix.
func (router *router) addPrefix(prefix string, verbe string, callback func(http.ResponseWriter, *http.Request)) {
	if _, exist := router.prefixes[prefix]; !exist {
		router.prefixes[prefix] = &route{path: prefix, handlers: make(map[string]func(http.ResponseWriter, *http.Request))}
	}
	if router.instrument {
		router.prefixes[prefix].handlers[verbe] = router.wrapHandlerWithMetrics(prefix, callback)
	} else {
		router.prefixes[prefix].handlers[verbe] = callback
	}
}

// wrapHandlerWithMetrics adds Prometheus instrumentation around the handler
func (router *router) wrapHandlerWithMetrics(handlerName string, h func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		router.metrics.WithLabelValues(handlerName, r.Method).Inc()
		h(w, r)
	}
}

func (router *router) count(handler, method string) {
	if router.instrument {
		router.metrics.WithLabelValues(handler, method).Inc()
	}
}

// handleMetrics exposes Prometheus metrics
func (router *router) handleMetrics(w http.ResponseWriter, r *http.Request) {
	promhttp.HandlerFor(router.registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}

func (router *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r, ok := router.routes[req.URL.Path]; ok {
		if handler, ok := r.handlers[req.Method]; ok {
			fmt.Println("ServeHTTP:", req.Method, req.URL.Path)
			handler(w, req)
			return
		}
		router.count(req.URL.Path, req.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	for prefix, r := range router.prefixes {
		if strings.HasPrefix(req.URL.Path, prefix) {
			if handler, ok := r.handlers[req.Method]; ok {
				handler(w, req)
				return
			}
			router.count(prefix, req.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
	http.NotFound(w, req)
}
