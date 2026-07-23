package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jvkassi/simple-html-app/backend/internal/cache"
	"github.com/jvkassi/simple-html-app/backend/internal/db"
)

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "notes_http_requests_total",
		Help: "Total HTTP requests handled by the notes backend.",
	}, []string{"route", "method", "status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "notes_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"route", "method"})

	cacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "notes_cache_hits_total",
		Help: "Notes served from the Redis cache.",
	})
	cacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "notes_cache_misses_total",
		Help: "Notes not found in the Redis cache, fetched from Postgres.",
	})
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := requireEnv(logger, "DATABASE_URL")
	redisAddr := requireEnv(logger, "REDIS_ADDR")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		logger.Error("postgres connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}

	rdb := cache.Connect(redisAddr)

	srv := newServer(pool, rdb, logger)

	go func() {
		logger.Info("backend listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}

func newServer(pool *pgxpool.Pool, rdb *cache.Cache, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(w, "postgres not ready", http.StatusServiceUnavailable)
			return
		}
		if err := rdb.Ping(r.Context()); err != nil {
			http.Error(w, "redis not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("GET /metrics", promhttp.Handler())

	mux.Handle("GET /api/notes", instrument("/api/notes", "GET", listNotesHandler(pool, logger)))
	mux.Handle("POST /api/notes", instrument("/api/notes", "POST", createNoteHandler(pool, rdb, logger)))
	mux.Handle("GET /api/notes/{id}", instrument("/api/notes/{id}", "GET", getNoteHandler(pool, rdb, logger)))

	return &http.Server{
		Addr:              ":8080",
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func requireEnv(logger *slog.Logger, name string) string {
	v := os.Getenv(name)
	if v == "" {
		logger.Error("missing required environment variable", "name", name)
		os.Exit(1)
	}
	return v
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func instrument(route, method string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		h(sw, r)
		httpDuration.WithLabelValues(route, method).Observe(time.Since(start).Seconds())
		httpRequests.WithLabelValues(route, method, strconv.Itoa(sw.status)).Inc()
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func listNotesHandler(pool *pgxpool.Pool, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notes, err := db.ListNotes(r.Context(), pool)
		if err != nil {
			logger.Error("list notes failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if notes == nil {
			notes = []db.Note{}
		}
		writeJSON(w, http.StatusOK, notes)
	}
}

func createNoteHandler(pool *pgxpool.Pool, rdb *cache.Cache, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Body string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Body) == "" {
			http.Error(w, "body is required", http.StatusBadRequest)
			return
		}

		note, err := db.CreateNote(r.Context(), pool, body.Body)
		if err != nil {
			logger.Error("create note failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if err := rdb.Invalidate(r.Context(), "notes:list"); err != nil {
			logger.Warn("cache invalidate failed", "error", err)
		}

		writeJSON(w, http.StatusCreated, note)
	}
}

func getNoteHandler(pool *pgxpool.Pool, rdb *cache.Cache, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		cacheKey := "notes:" + strconv.FormatInt(id, 10)
		if cached, ok, err := rdb.Get(r.Context(), cacheKey); err == nil && ok {
			cacheHits.Inc()
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "hit")
			w.Write([]byte(cached))
			return
		}
		cacheMisses.Inc()

		note, err := db.GetNote(r.Context(), pool, id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		payload, err := json.Marshal(note)
		if err == nil {
			if err := rdb.Set(r.Context(), cacheKey, string(payload), 30*time.Second); err != nil {
				logger.Warn("cache set failed", "error", err)
			}
		}

		writeJSON(w, http.StatusOK, note)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
