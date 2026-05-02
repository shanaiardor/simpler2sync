package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"simpler2sync/internal/config"
	"simpler2sync/internal/gui"
	"simpler2sync/internal/r2client"
	"simpler2sync/internal/scheduler"
	"simpler2sync/internal/store"
	"simpler2sync/internal/sync"
)

var (
	cfg   *config.AppConfig
	app   *gui.GUIApp
	r2    *r2client.Client
	db    *store.Store
	sched *scheduler.Scheduler
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var err error
	cfg, err = config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	log.Printf("Config: %s", config.ConfigPath())

	db, err = store.Open(config.ConfigPathDir())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer db.Close()

	if cfg.R2.Endpoint != "" {
		r2, err = r2client.New(cfg.R2.Endpoint, cfg.R2.AccessKeyID, cfg.R2.SecretAccessKey, cfg.R2.Region)
		if err != nil {
			log.Printf("WARNING: R2 client init failed: %v", err)
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	app = gui.NewGUIApp("simpler2sync", func() {
		log.Println("shutting down...")
		if sched != nil {
			sched.Stop()
		}
		cfg.Save()
	})

	app.Init(cfg, func(task *config.SyncTask) {
		go doSync(task)
	})

	startScheduler()

	go func() {
		<-sigCh
		app.Quit()
	}()

	app.Run()
	return nil
}

func doSync(task *config.SyncTask) {
	if r2 == nil {
		app.Log("ERROR: R2 client not configured")
		return
	}
	app.Log(fmt.Sprintf("Starting sync: %s", task.Name))
	result, err := sync.RunSync(
		scheduler.BackgroundContext(),
		r2, db,
		task.Name, task.LocalPath, task.RemoteBucket, task.RemotePrefix,
		cfg.Settings.ConflictStrategy,
		func(action sync.Action, progress, total int, e error) {
			if e != nil {
				app.Log(fmt.Sprintf("[%d/%d] ERROR %s %s: %v", progress, total, action.Type, action.LocalPath, e))
			} else {
				app.Log(fmt.Sprintf("[%d/%d] %s %s", progress, total, action.Type, action.LocalPath))
			}
		},
	)
	if err != nil {
		app.Log(fmt.Sprintf("Sync %s failed: %v", task.Name, err))
	} else {
		app.Log(fmt.Sprintf("Sync %s complete: %d success, %d failed", task.Name, result.Success, result.Failed))
	}
}

func startScheduler() {
	if cfg.Settings.CronExpression != "" {
		sched = scheduler.New(func() {
			app.Log("Scheduled sync triggered")
			for _, task := range cfg.Tasks {
				if task.Enabled {
					doSync(&task)
				}
			}
		})
		sched.StartCron(cfg.Settings.CronExpression)
		app.Log(fmt.Sprintf("Scheduler started with cron: %s", cfg.Settings.CronExpression))
	} else if cfg.Settings.IntervalSeconds > 0 {
		sched = scheduler.New(func() {
			app.Log("Scheduled sync triggered")
			for _, task := range cfg.Tasks {
				if task.Enabled {
					doSync(&task)
				}
			}
		})
		sched.StartInterval(cfg.Settings.IntervalSeconds)
		app.Log(fmt.Sprintf("Scheduler started with interval: %ds", cfg.Settings.IntervalSeconds))
	}
}
