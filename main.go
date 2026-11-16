package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"ruangku-backend/app/repository"
	"ruangku-backend/app/service"
	"ruangku-backend/config"
	"ruangku-backend/database"
	"ruangku-backend/helper"
	"ruangku-backend/route"
)

const (
	minJWTSecretLength = 32
	shutdownTimeout    = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config.LoadEnv()

	loc, err := time.LoadLocation(config.GetEnv("APP_TIMEZONE", "Asia/Jakarta"))
	if err != nil {
		return err
	}
	time.Local = loc

	secret := config.GetEnv("JWT_SECRET", "")
	if len(secret) < minJWTSecretLength {
		return errors.New("JWT_SECRET harus diisi minimal 32 karakter")
	}
	jwtManager := helper.NewJWTManager(secret, time.Duration(config.GetEnvInt("JWT_TTL_MINUTES", 60))*time.Minute)

	pool, err := database.NewPool(context.Background())
	if err != nil {
		return err
	}
	defer pool.Close()

	txManager := repository.NewTxManager(pool)

	userRepo := repository.NewUserRepository(pool)
	roomRepo := repository.NewRoomRepository(pool)
	bookingRepo := repository.NewBookingRepository(pool)
	historyRepo := repository.NewHistoryRepository(pool)

	authService := service.NewAuthService(userRepo, jwtManager)
	userService := service.NewUserService(userRepo)
	roomService := service.NewRoomService(roomRepo)
	bookingService := service.NewBookingService(bookingRepo, roomRepo, txManager)
	historyService := service.NewHistoryService(historyRepo)

	err = authService.EnsureAdmin(
		context.Background(),
		config.GetEnv("ADMIN_NAME", "Admin"),
		config.GetEnv("ADMIN_EMAIL", ""),
		config.GetEnv("ADMIN_PASSWORD", ""),
	)
	if err != nil {
		return err
	}

	app := config.NewApp()
	route.Register(app, route.Dependencies{
		Pool:    pool,
		JWT:     jwtManager,
		Auth:    authService,
		User:    userService,
		Room:    roomService,
		Booking: bookingService,
		History: historyService,
	})

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- app.Listen(":" + config.GetEnv("APP_PORT", "8080"))
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case <-stop:
		log.Println("menghentikan server...")
		return app.ShutdownWithTimeout(shutdownTimeout)
	}
}
