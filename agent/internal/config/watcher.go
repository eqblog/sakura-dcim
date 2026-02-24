package config

import (
	"sync"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

type Watcher struct {
	path     string
	logger   *zap.Logger
	mu       sync.RWMutex
	current  *Config
	onChange func(*Config)
}

func NewWatcher(path string, initial *Config, logger *zap.Logger) *Watcher {
	return &Watcher{
		path:    path,
		logger:  logger,
		current: initial,
	}
}

func (w *Watcher) OnChange(fn func(*Config)) {
	w.onChange = fn
}

func (w *Watcher) Current() *Config {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.current
}

func (w *Watcher) Watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		w.logger.Error("failed to create file watcher", zap.Error(err))
		return
	}
	defer watcher.Close()

	if err := watcher.Add(w.path); err != nil {
		w.logger.Error("failed to watch config file", zap.Error(err), zap.String("path", w.path))
		return
	}

	w.logger.Info("watching config file for changes", zap.String("path", w.path))

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				w.logger.Info("config file changed, reloading")
				newCfg, err := Load(w.path)
				if err != nil {
					w.logger.Error("failed to reload config", zap.Error(err))
					continue
				}
				w.mu.Lock()
				w.current = newCfg
				w.mu.Unlock()
				if w.onChange != nil {
					w.onChange(newCfg)
				}
				w.logger.Info("config reloaded successfully")
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			w.logger.Error("file watcher error", zap.Error(err))
		}
	}
}
