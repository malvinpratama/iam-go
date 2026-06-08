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

	userv1 "github.com/malvin/iam-go/gen/user/v1"
	"github.com/malvin/iam-go/pkg/config"
	"github.com/malvin/iam-go/pkg/db"
	"github.com/malvin/iam-go/pkg/interceptor"
	"github.com/malvin/iam-go/pkg/logger"
	"github.com/malvin/iam-go/pkg/migrate"
	user "github.com/malvin/iam-go/services/user"
	"github.com/malvin/iam-go/services/user/internal/handler"
)

func main() {
	log := logger.New("user")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbURL := config.MustEnv("USER_DATABASE_URL")
	port := config.Getenv("USER_GRPC_PORT", "50052")

	sub, err := fs.Sub(user.MigrationsFS, "db/migrations")
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

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Error("listen", "err", err)
		os.Exit(1)
	}

	srv := grpc.NewServer(interceptor.Chain(config.InternalToken()))
	userv1.RegisterUserServiceServer(srv, handler.New(pool))

	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(srv, hs)
	if !config.IsProduction() {
		reflection.Register(srv) // dev only
	}

	go func() {
		<-ctx.Done()
		log.Info("shutting down")
		srv.GracefulStop()
	}()

	log.Info("user service listening", "port", port)
	if err := srv.Serve(lis); err != nil {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
}
