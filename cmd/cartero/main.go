package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cartero/internal/config"
)

var (
	configPath = flag.String("config", "config.toml", "Path to configuration file")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %v\n", sig)
		fmt.Println("Shutting down gracefully...")
		cancel()
	}()

	if err := run(ctx); err != nil {
		log.Fatalf("Fatal error: %v", err)
	}
}

func run(ctx context.Context) error {
	fmt.Printf("Loading configuration from: %s\n", *configPath)

	bot, err := config.LoadAndBuild(ctx, *configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	fmt.Printf("Starting bot: %s\n", bot.Name())

	errChan := make(chan error, 1)
	go func() {
		if err := bot.Start(ctx); err != nil && err != context.Canceled {
			errChan <- err
		}
		close(errChan)
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		fmt.Println("\nInitiating shutdown...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := bot.Stop(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}
	}

	fmt.Println("Bot stopped successfully")
	return nil
}
