package engine

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/anacrolix/game"
	"github.com/fsnotify/fsnotify"
)

func (e *Engine) isTaskInList(ih string) bool {
	e.RLock()
	defer e.RUnlock()
	_, ok := e.ts[ih]
	return ok
}

func (e *Engine) upsertGame(ih, name string, isQueueing bool) (*Game, error) {
	defer func() {
		e.TsChanged <- struct{}{}
	}()

	e.RLock()
	game, ok := e.ts[ih]
	e.RUnlock()
	if !ok {
		game = &Game{
			Name:       name,
			InfoHash:   ih,
			IsQueueing: isQueueing,
			AddedAt:    time.Now(),
			cld:        e.cld,
			e:          e,
			dropWait:   make(chan struct{}),
		}
		e.Lock()
		e.ts[ih] = game
		e.Unlock()
		return game, nil
	}
	game.IsQueueing = isQueueing
	return game, ErrTaskExists
}

func (e *Engine) getGame(infohash string) (*Game, error) {
	if t, ok := e.ts[infohash]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("Missing game %x", infohash)
}

func (e *Engine) deleteGame(infohash string) {
	delete(e.ts, infohash)
	e.TsChanged <- struct{}{}
}

func (e *Engine) ParseTrackerList() error {
	conf := e.config.TrackerList

	trackers := []string{}

	for _, l := range strings.Split(conf, "\n") {
		line := strings.TrimSpace(l)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "remote:") {
			if lst, err := fetchTxtList(line[7:]); err == nil {
				trackers = append(trackers, lst...)
			} else {
				log.Println("[ParseTrackerList] ignored", err, line)
			}
		} else {
			trackers = append(trackers, line)
		}
	}

	// remove duplicated entries
	dupMap := make(map[string]struct{})
	for _, t := range trackers {
		dupMap[t] = struct{}{}
	}
	for t := range dupMap {
		e.Trackers = append(e.Trackers, t)
	}

	log.Printf("[ParseTrackerList] got %d trackers", len(e.Trackers))
	return nil
}

func (e *Engine) WriteStauts(_w io.Writer) {
	e.RLock()
	defer e.RUnlock()
	if e.client != nil {
		e.client.WriteStatus(_w)
	}
}

func (e *Engine) ConnStat() game.ConnStats {
	e.RLock()
	defer e.RUnlock()
	if e.client != nil {
		return e.client.ConnStats()
	}
	return game.ConnStats{}
}

func (e *Engine) StartGameWatcher() error {

	if e.watcher != nil {
		log.Println("Game Watcher: close")
		e.watcher.Close()
		e.watcher = nil
	}

	if w, err := os.Stat(e.config.WatchDirectory); os.IsNotExist(err) || (err == nil && !w.IsDir()) {
		return fmt.Errorf("[Watcher] WatchDirectory [%s] is not a dir, will not watch", e.config.WatchDirectory)
	}

	log.Printf("Game Watcher: watching game file in %s", e.config.WatchDirectory)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	e.watcher = watcher

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					if !strings.HasSuffix(event.Name, ".game") {
						continue
					}
					if st, err := os.Stat(event.Name); err != nil {
						log.Println(err)
						continue
					} else if st.IsDir() {
						continue
					}

					if err := e.NewGameByFilePath(event.Name); err == nil {
						log.Printf("Game Watcher: added %s, file removed\n", event.Name)
						os.Remove(event.Name)
					} else {
						log.Printf("Game Watcher: fail to add %s, ERR:%#v\n", event.Name, err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()
	err = watcher.Add(e.config.WatchDirectory)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}
