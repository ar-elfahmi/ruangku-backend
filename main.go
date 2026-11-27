package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	_ "ruangku-backend/apidocs"
	"ruangku-backend/app/repository"
	"ruangku-backend/app/service"
	"ruangku-backend/config"
	"ruangku-backend/database"
	"ruangku-backend/helper"
	"ruangku-backend/queue"
	"ruangku-backend/route"
)

const shutdownTimeout = 10 * time.Second

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
	refreshSecret := config.GetEnv("JWT_REFRESH_SECRET", "")
	if err := helper.ValidateJWTSecrets(secret, refreshSecret); err != nil {
		return err
	}
	jwtManager := helper.NewJWTManager(secret, time.Duration(config.GetEnvInt("JWT_TTL_MINUTES", 15))*time.Minute)
	refreshManager := helper.NewRefreshJWTManager(
		refreshSecret,
		time.Duration(config.GetEnvInt("JWT_REFRESH_TTL_HOURS", 168))*time.Hour,
	)

	pool, err := database.NewPool(context.Background())
	if err != nil {
		return err
	}
	defer pool.Close()

	mongoDB, err := database.NewMongo(
		context.Background(),
		config.GetEnv("MONGO_URI", "mongodb://127.0.0.1:27017"),
		config.GetEnv("MONGO_DB", "ruangku"),
	)
	if err != nil {
		return err
	}

	producer := queue.NewProducer(queue.LoadConfig())
	defer producer.Close()

	txManager := repository.NewTxManager(pool)

	userRepo := repository.NewUserRepository(pool)
	roomRepo := repository.NewRoomRepository(pool)
	bookingRepo := repository.NewBookingRepository(pool)
	historyRepo := repository.NewHistoryRepository(pool)
	refreshRepo := repository.NewRefreshRepository(pool)
	notificationRepo, err := repository.NewNotificationRepository(mongoDB)
	if err != nil {
		return err
	}

	authService := service.NewAuthService(
		userRepo,
		jwtManager,
		service.WithRefreshIssuer(service.NewRefreshIssuer(refreshRepo, txManager, refreshManager)),
	)
	sessionService := service.NewSessionService(refreshRepo, refreshManager, jwtManager, userRepo, txManager)
	userService := service.NewUserService(userRepo)
	roomService := service.NewRoomService(roomRepo)
	bookingService := service.NewBookingService(bookingRepo, roomRepo, txManager, service.WithNotifier(producer))
	historyService := service.NewHistoryService(historyRepo)
	notificationService := service.NewNotificationService(notificationRepo)

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
		Pool:         pool,
		JWT:          jwtManager,
		Auth:         authService,
		Session:      sessionService,
		User:         userService,
		Room:         roomService,
		Booking:      bookingService,
		History:      historyService,
		Notification: notificationService,
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
