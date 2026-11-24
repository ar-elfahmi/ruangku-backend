package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"ruangku-backend/app/repository"
	"ruangku-backend/config"
	"ruangku-backend/database"
	"ruangku-backend/queue"
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

	mongoDB, err := database.NewMongo(
		context.Background(),
		config.GetEnv("MONGO_URI", "mongodb://127.0.0.1:27017"),
		config.GetEnv("MONGO_DB", "ruangku"),
	)
	if err != nil {
		return err
	}

	store, err := repository.NewNotificationRepository(mongoDB)
	if err != nil {
		return err
	}

	worker := queue.NewWorker(queue.LoadConfig(), store)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		log.Println("menghentikan worker...")
		worker.Shutdown()
	}()

	log.Println("worker notifikasi berjalan")
	return worker.Run()
}
