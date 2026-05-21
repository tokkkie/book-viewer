package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx         context.Context
	rootPath    string
	mu          sync.Mutex
	tracksReady chan struct{}
	domReady    chan struct{}
}

func NewApp() *App {
	return &App{
		tracksReady: make(chan struct{}),
		domReady:    make(chan struct{}),
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	go func() {
		config, err := loadConfig()
		if err != nil {
			fmt.Printf("Config load error: %v\n", err)
		} else if config.RootDirectory != "" {
			fmt.Printf("Loading root directory: %s\n", config.RootDirectory)
			a.SetRootDirectory(config.RootDirectory)
		} else {
			fmt.Println("No root directory configured")
		}
		close(a.tracksReady)
	}()

	go func() {
		fmt.Println("Waiting for tracksReady...")
		<-a.tracksReady
		fmt.Println("tracksReady received, waiting for domReady...")
		<-a.domReady
		fmt.Println("domReady received, emitting dataReady event")
		runtime.EventsEmit(a.ctx, "dataReady", true)
		fmt.Println("dataReady event emitted")
	}()
}

func (a *App) OnDomReady(ctx context.Context) {
	fmt.Println("OnDomReady called")
	close(a.domReady)
}

func (a *App) SetRootDirectory(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.rootPath = path

	config, _ := loadConfig()
	config.RootDirectory = path
	if err := saveConfig(config); err != nil {
		fmt.Printf("Warning: failed to save config: %v\n", err)
	}

	return nil
}

func (a *App) GetRootDirectory() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.rootPath
}

func (a *App) OpenDirectoryDialog() (string, error) {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Root Directory",
	})
	return path, err
}

func (a *App) SaveLastSelection(series, volume string) error {
	config, err := loadConfig()
	if err != nil {
		config = &Config{}
	}
	config.LastSeries = series
	config.LastVolume = volume

	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save selection: %w", err)
	}
	return nil
}

func (a *App) GetLastSelection() map[string]string {
	config, err := loadConfig()
	if err != nil {
		return map[string]string{"series": "", "volume": ""}
	}
	return map[string]string{
		"series": config.LastSeries,
		"volume": config.LastVolume,
	}
}
