package main

import (
	"context"
	"io/fs"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	authv1 "github.com/malvin/iam-go/gen/auth/v1"
	"github.com/malvin/iam-go/pkg/config"
	"github.com/malvin/iam-go/pkg/db"
	"github.com/malvin/iam-go/pkg/email"
	"github.com/malvin/iam-go/pkg/events"
	"github.com/malvin/iam-go/pkg/interceptor"
	"github.com/malvin/iam-go/pkg/jwt"
	"github.com/malvin/iam-go/pkg/logger"
	"github.com/malvin/iam-go/pkg/migrate"
	auth "github.com/malvin/iam-go/services/auth"
	authdb "github.com/malvin/iam-go/services/auth/internal/db"
	"github.com/malvin/iam-go/services/auth/internal/handler"
	"github.com/malvin/iam-go/services/auth/internal/outbox"
)

func main() {
	log := logger.New("auth")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := config.ValidateSecurity(); err != nil {
		log.Error("insecure configuration", "err", err)
		os.Exit(1)
	}

	dbURL := config.MustEnv("AUTH_DATABASE_URL")
	port := config.Getenv("AUTH_GRPC_PORT", "50051")

	sub, err := fs.Sub(auth.MigrationsFS, "db/migrations")
	if err != nil {
		log.Error("embed migrations", "err", err)
		os.Exit(1)
	}
	if err := migrate.Run(ctx, dbURL, sub); err != nil {
		log.Error("run migrations", "err", err)
		os.Exit(1)
	}
	log.Info("migrations applied")

	pool, err := db.NewPool(ctx, dbURL)
	if err != nil {
		log.Error("connect db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := handler.BootstrapAdmin(ctx, pool,
		config.Getenv("BOOTSTRAP_ADMIN_EMAIL", ""),
		config.Getenv("BOOTSTRAP_ADMIN_PASSWORD", ""),
	); err != nil {
		log.Error("bootstrap admin", "err", err)
		os.Exit(1)
	}

	jwtCfg := config.LoadJWT()
	h := handler.New(pool, jwt.NewManager(jwtCfg), jwtCfg.RefreshTTL, email.NewLogSender(log))

	// Outbox relay: drain pending domain events to NATS JetStream. Optional —
	// without NATS_URL the events are still recorded; the gateway's lazy profile
	// healing keeps the system working.
	if url := config.NatsURL(); url != "" {
		nc, js, err := events.Connect(url)
		if err != nil {
			log.Error("connect nats", "err", err)
			os.Exit(1)
		}
		defer nc.Close()
		if err := events.EnsureStream(js); err != nil {
			log.Error("ensure stream", "err", err)
			os.Exit(1)
		}
		go outbox.NewRelay(authdb.New(pool), js, log).Run(ctx)
		log.Info("outbox relay started", "nats", url)
	} else {
		log.Warn("NATS_URL not set — event publishing disabled")
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Error("listen", "err", err)
		os.Exit(1)
	}

	srv := grpc.NewServer(interceptor.Chain(config.InternalToken()))
	authv1.RegisterAuthServiceServer(srv, h)

	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(srv, hs)
	if !config.IsProduction() {
		reflection.Register(srv) // dev only — avoid schema disclosure in prod
	}

	go func() {
		<-ctx.Done()
		log.Info("shutting down")
		srv.GracefulStop()
	}()

	log.Info("auth service listening", "port", port)
	if err := srv.Serve(lis); err != nil {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
}
