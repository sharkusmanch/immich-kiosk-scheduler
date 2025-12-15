// Package main is the entry point for immich-kiosk-scheduler.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/sharkusmanch/immich-kiosk-scheduler/internal/config"
	"github.com/sharkusmanch/immich-kiosk-scheduler/internal/scheduler"
	"github.com/sharkusmanch/immich-kiosk-scheduler/internal/server"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

var (
	cfgFile  string
	port     int
	logLevel string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "immich-kiosk-scheduler",
	Short: "A scheduling proxy for Immich Kiosk albums",
	Long: `immich-kiosk-scheduler is a lightweight service that redirects requests
to Immich Kiosk with the appropriate album based on a date-based schedule.

It supports seasonal album rotation (e.g., Christmas album from Nov 15 to Jan 1)
without requiring any changes to your kiosk configuration.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, buildDate),
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long:  `Start the HTTP server that handles redirects to Immich Kiosk.`,
	RunE:  runServe,
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the schedule for a specific date",
	Long: `Test which album would be selected for a specific date.
This is useful for verifying your schedule configuration.`,
	RunE: runTest,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (default: ./config.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")

	// Bind to env vars
	_ = viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	_ = viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))

	// Serve command flags
	serveCmd.Flags().IntVar(&port, "port", 8080, "port to listen on")
	_ = viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))

	// Test command flags
	testCmd.Flags().String("date", "", "date to test (MM-DD format, defaults to today)")

	// Register commands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(testCmd)
}

func initConfig() {
	viper.SetEnvPrefix("IKS")
	viper.AutomaticEnv()

	// Check for config file in env var
	if cfgFile == "" {
		cfgFile = viper.GetString("config")
	}
}

func setupLogger(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))
}

func runServe(cmd *cobra.Command, args []string) error {
	setupLogger(viper.GetString("log_level"))

	if cfgFile == "" {
		cfgFile = "config.yaml"
	}

	slog.Info("loading configuration", slog.String("file", cfgFile))

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override port from CLI/env if set
	if viper.IsSet("port") {
		cfg.Port = viper.GetInt("port")
	}

	sched, err := scheduler.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create scheduler: %w", err)
	}

	slog.Info("scheduler initialized",
		slog.Int("schedules", sched.GetScheduleCount()),
		slog.String("current_schedule", sched.GetCurrentScheduleName()),
		slog.String("current_album", sched.GetCurrentAlbum()),
	)

	srv, err := server.New(cfg, sched)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("received shutdown signal")
		cancel()
	}()

	return srv.StartWithContext(ctx)
}

func runTest(cmd *cobra.Command, args []string) error {
	setupLogger("info")

	if cfgFile == "" {
		cfgFile = "config.yaml"
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	sched, err := scheduler.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create scheduler: %w", err)
	}

	// Parse date flag
	dateStr, _ := cmd.Flags().GetString("date")
	var testDate time.Time

	if dateStr == "" {
		testDate = time.Now()
		fmt.Printf("Testing schedule for today (%s)\n\n", testDate.Format("January 2"))
	} else {
		month, day, err := scheduler.ParseMonthDay(dateStr)
		if err != nil {
			return fmt.Errorf("invalid date format: %w", err)
		}
		testDate = time.Date(time.Now().Year(), time.Month(month), day, 0, 0, 0, 0, time.Local)
		fmt.Printf("Testing schedule for %s\n\n", testDate.Format("January 2"))
	}

	album := sched.GetAlbumForDate(testDate)
	scheduleName := sched.GetScheduleNameForDate(testDate)

	fmt.Printf("Schedule:  %s\n", scheduleName)
	fmt.Printf("Album ID:  %s\n", album)
	fmt.Printf("Redirect:  %s?album=%s\n", cfg.KioskURL, album)

	return nil
}
