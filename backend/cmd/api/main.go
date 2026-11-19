package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/internal/api"
	"backend/internal/config"
	"backend/internal/forum"
	"backend/internal/kb"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	if cfg.BrowserClearance {
		if path := forum.FindBrowser(cfg.BrowserPath); path != "" {
			log.Printf("Браузер для проверки форума: %s", path)
		} else {
			log.Printf("Chrome/Chromium/Edge не найден: если форум включит проверку браузера, укажите путь в FORUM_BROWSER_PATH")
		}
	}

	fetcher := forum.NewFetcher(forum.FetcherOptions{
		UserAgent:        cfg.UserAgent,
		Delay:            cfg.ScrapeDelay,
		Retries:          2,
		BrowserClearance: cfg.BrowserClearance,
		BrowserPath:      cfg.BrowserPath,
		ForumBaseURL:     cfg.ForumBaseURL,
	})

	servers := make([]kb.Server, 0, len(cfg.Servers))
	for _, s := range cfg.Servers {
		servers = append(servers, kb.Server{Code: s.Code, Name: s.Name, Sources: s.Sources})
	}

	store := kb.NewStore(kb.Options{
		BaseURL:             cfg.ForumBaseURL,
		GeneralSources:      cfg.GeneralSources,
		Servers:             servers,
		CacheFile:           cfg.CacheFile,
		CacheMaxAge:         cfg.CacheMaxAge,
		MaxThreadsPerSource: cfg.MaxThreadsPerSource,
		ForumPages:          cfg.ForumPages,
		MaxPagesPerThread:   cfg.MaxPagesPerThread,
	}, fetcher)

	if err := store.LoadCache(); err != nil {
		log.Printf("Кэш не загружен: %v", err)
	} else if st := store.Status(); st.Ready {
		log.Printf("Загружен кэш: документов %d, пунктов правил %d", st.Documents, st.Rules)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		next := store.LastRefresh().Add(cfg.RefreshInterval)

		for {
			if wait := time.Until(next); wait > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(wait):
				}
			}

			if err := store.Refresh(ctx); err != nil && ctx.Err() == nil {
				log.Printf("Ошибка обновления: %v", err)
			}

			if ctx.Err() != nil {
				return
			}

			next = time.Now().Add(cfg.RefreshInterval)
			if store.Status().Blocked {
				next = time.Now().Add(15 * time.Minute)
			}
		}
	}()

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", api.NewHandler(store, fetcher).Register)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 40 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("API запущен на порту %s", cfg.Port)

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Ошибка HTTP-сервера: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Остановка API...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Ошибка остановки: %v", err)
	}
}
