package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpadapter "coffee-pos/backend/internal/adapters/http"
	"coffee-pos/backend/internal/adapters/security"
	appauth "coffee-pos/backend/internal/app/auth"
	"coffee-pos/backend/internal/config"
)

func main() {
	os.Exit(run(context.Background(), os.Args, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "usage: coffee-pos serve | coffee-pos auth hash-pin <pin>")
		return 2
	}

	switch args[1] {
	case "serve":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "usage: coffee-pos serve")
			return 2
		}
		if err := runServe(ctx); err != nil {
			slog.Error("backend server failed", "error", err)
			return 1
		}
		return 0
	case "auth":
		if len(args) == 4 && args[2] == "hash-pin" {
			if err := runHashPIN(stdout, args[3]); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			return 0
		}
		fmt.Fprintln(stderr, "usage: coffee-pos auth hash-pin <pin>")
		return 2
	default:
		fmt.Fprintln(stderr, "usage: coffee-pos serve | coffee-pos auth hash-pin <pin>")
		return 2
	}
}

func runServe(_ context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("backend configuration failed", "error", err)
		return err
	}

	authService := appauth.NewService(appauth.Dependencies{
		CashierPINHash: cfg.CashierPINHash,
		Verifier:       security.BcryptPINHash{},
		Sessions:       security.NewInMemorySessionStore(),
		RateLimiter:    security.NewInMemoryRateLimiter(),
		SessionIDs:     security.RandomSessionIDGenerator{},
		Clock:          systemClock{},
		Location:       cfg.BusinessLocation,
	})

	server := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpadapter.NewRouter(httpadapter.RouterOptions{
			AuthService: authService,
			Cookie: httpadapter.CookieConfig{
				Name:     cfg.SessionCookieName,
				Path:     "/",
				Secure:   cfg.SessionCookieSecure,
				SameSite: http.SameSiteLaxMode,
			},
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting coffee POS backend", "addr", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-signalCh:
		slog.Info("shutting down coffee POS backend", "signal", sig.String())
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("backend shutdown failed", "error", err)
			return err
		}
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	return nil
}

func runHashPIN(stdout io.Writer, pin string) error {
	hash, err := (security.BcryptPINHash{}).HashPIN(pin)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout, hash)
	return err
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}
