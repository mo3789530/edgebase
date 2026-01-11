package shutdown

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgebase/platform/control-plane/internal/logger"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Manager struct {
	app   *fiber.App
	db    *gorm.DB
	hooks []func(context.Context) error
}

func NewManager(app *fiber.App, db *gorm.DB) *Manager {
	return &Manager{
		app:   app,
		db:    db,
		hooks: make([]func(context.Context) error, 0),
	}
}

// AddHook adds a cleanup hook
func (m *Manager) AddHook(hook func(context.Context) error) {
	m.hooks = append(m.hooks, hook)
}

// Start listens for shutdown signals and gracefully shuts down
func (m *Manager) Start() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		logger.Info("", "shutdown_signal_received", map[string]interface{}{
			"signal": sig.String(),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Run hooks first (e.g. flush metrics before closing DB/App?)
		// Usually app shutdown first (stop accepting requests), then hooks, then DB.
		
		// Shutdown Fiber app
		if err := m.app.ShutdownWithContext(ctx); err != nil {
			logger.Error("", "fiber_shutdown_error", err)
		}
		
		// Run hooks
		for _, hook := range m.hooks {
			if err := hook(ctx); err != nil {
				logger.Error("", "shutdown_hook_error", err)
			}
		}

		// Close database connection
		if sqlDB, err := m.db.DB(); err == nil {
			sqlDB.Close()
			logger.Info("", "database_connection_closed", nil)
		}

		logger.Info("", "graceful_shutdown_complete", nil)
		os.Exit(0)
	}()
}
