package main

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"os"

	mw "bitbucket.org/coveord/blitz-2018/server/web/middleware"
	"github.com/inconshreveable/log15"
	"github.com/pressly/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/thoas/stats"
	"golang.org/x/oauth2"
	oauthGH "golang.org/x/oauth2/github"
	// "github.com/graphql-go/graphql"
)

const DefaultPort = ":3000"

func main() {
	// Dependency Injection and initialization
	r := chi.NewRouter()
	stats := stats.New()
	logger := log15.Root()
	prometheusRegistry := prometheus.NewRegistry()

	prometheusRegistry.MustRegister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheusRegistry.MustRegister(prometheus.NewGoCollector())

	// Setup middleware global and filters
	r.Use(
		// The order has importance
		stats.Handler,
		mw.Prometheus(prometheusRegistry),
		mw.RequestID,
		mw.RequestLogger(logger),
		mw.Recoverer,
		cors.Default().Handler,
	)

	githubOauthConfig := &oauth2.Config{
		ClientID:     "GITHUB_CLIENT_ID",
		ClientSecret: "GITHUB_CLIENT_SECRET",
		Endpoint:     oauthGH.Endpoint,
		Scopes:       []string{"user:email"},
		RedirectURL:  "http://localhost:3000/login/callback/github",
	}

	r.Get("/", homeHandler)
	r.Get("/stats", statsHandler(stats))
	r.Get("/healthz", healthHandler)
	r.Mount("/debug", mw.Profiler())
	r.Mount("/metrics", promhttp.HandlerFor(prometheusRegistry, promhttp.HandlerOpts{}))

	port := DefaultPort
	if p := os.Getenv("PORT"); p != "" {
		port = ":" + p
	}

	logger.Info("Server starting", "port", port)
	http.ListenAndServe(port, r)
	// FIXME: Implement graceful shutdown
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	logger := mw.GetLogger(r)
	w.WriteHeader(http.StatusOK)

	if err := homeTmpl.Execute(w, struct{}{}); err != nil {
		logger.Error("Error parsing template `homeTmpl`", "err", err)
	}
}

var homeTmpl = template.Must(template.New("index").Parse(`<html>
<head>
<title>/home</title>
</head>
<body>
<h1>Blitz Home</h1>

<a href="/login/github">Login with github</a>
`))

func statsHandler(s *stats.Stats) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		PrettyJSONEncoder(w).Encode(s.Data())
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// FIXME: Do a real health check
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))
}

func PrettyJSONEncoder(w io.Writer) *json.Encoder {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc
}
