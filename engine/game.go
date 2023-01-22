package engine

import (
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/anacrolix/game"
)

type Game struct {
	sync.Mutex

	//anacrolix/game
	InfoHash   string
	Name       string
	Magnet     string
	Loaded     bool
	Downloaded int64
	Uploaded   int64
	Size       int64
	Files      []*File

	//cloud game
	Stats          *game.GameStats
	Started        bool
	Done           bool
	DoneCmdCalled  bool
	IsQueueing     bool
	IsSeeding      bool
	ManualStarted  bool
	IsAllFilesDone bool
	Percent        float32
	DownloadRate   float32
	UploadRate     float32
	SeedRatio      float32
	AddedAt        time.Time
	StartedAt      time.Time
	FinishedAt     time.Time
	StoppedAt      time.Time
	updatedAt      time.Time
	t              *game.Game
	e              *Engine
	dropWait       chan struct{}
	cld            Server
}

type File struct {
	//anacrolix/game
	Path          string
	Size          int64
	Completed     int64
	Done          bool
	DoneCmdCalled bool
	//cloud game
	Started bool
	Percent float32
	f       *game.File
}

// Update retrive info from game.Game
func (game *Game) updateOnGotInfo(t *game.Game) {

	if t.Info() != nil && !game.Loaded {
		game.t = t
		game.Name = t.Name()
		game.Loaded = true
		game.updateFileStatus()
		game.updateGameStatus()
		game.updateConnStat()

		if game.Magnet == "" {
			meta := t.Metainfo()
			if ifo, err := meta.UnmarshalInfo(); err == nil {
				magnet := meta.Magnet(nil, &ifo).String()
				game.Magnet = magnet
			} else {
				game.Magnet = "ERROR{}"
			}
			game.Name = t.Name()
		}
	}
}

func (game *Game) updateConnStat() {
	now := time.Now()
	lastStat := game.Stats
	curStat := game.t.Stats()

	if lastStat == nil {
		game.updatedAt = now
		game.Stats = &curStat
		return
	}

	bRead := curStat.BytesReadUsefulData.Int64()
	bWrite := curStat.BytesWrittenData.Int64()

	lRead := lastStat.BytesReadUsefulData.Int64()
	lWrite := lastStat.BytesWrittenData.Int64()

	// download will stop if game is done (bRead equals)
	if bRead >= lRead || bWrite > lWrite {

		// calculate ratio
		if bRead > 0 {
			game.SeedRatio = float32(bWrite) / float32(bRead)
		} else if game.Done {
			game.SeedRatio = float32(bWrite) / float32(game.Size)
		}

		if lastStat != nil {
			// calculate rate
			dtinv := float32(time.Second) / float32(now.Sub(game.updatedAt))

			dldb := float32(bRead - lRead)
			game.DownloadRate = dldb * dtinv

			uldb := float32(bWrite - lWrite)
			game.UploadRate = uldb * dtinv
		}

		game.Downloaded = game.t.BytesCompleted()
		game.Uploaded = bWrite
		game.updatedAt = now
		game.Stats = &curStat
	}
}

func (game *Game) updateFileStatus() {
	if game.IsAllFilesDone {
		return
	}

	tfiles := game.t.Files()
	if len(tfiles) > 0 && game.Files == nil {
		game.Files = make([]*File, len(tfiles))
	}

	//merge in files
	doneFlag := true
	for i, f := range tfiles {
		path := f.Path()
		file := game.Files[i]
		if file == nil {
			file = &File{Path: path, Started: game.Started, f: f}
			game.Files[i] = file
		}

		file.Size = f.Length()
		file.Completed = f.BytesCompleted()
		file.Percent = percent(file.Completed, file.Size)
		file.Done = (file.Completed == file.Size)
		if file.Done && !file.DoneCmdCalled {
			file.DoneCmdCalled = true
			go game.callDoneCmd(file.Path, "file", file.Size)
		}
		if !file.Done {
			doneFlag = false
		}
	}

	game.IsAllFilesDone = doneFlag
}

func (game *Game) updateGameStatus() {
	game.Size = game.t.Length()
	game.Percent = percent(game.t.BytesCompleted(), game.Size)
	game.Done = (game.t.BytesMissing() == 0)
	game.IsSeeding = game.t.Seeding() && game.Done

	// this process called at least on second Update calls
	if game.Done && !game.DoneCmdCalled {
		game.DoneCmdCalled = true
		game.FinishedAt = time.Now()
		log.Println("[TaskFinished]", game.InfoHash)
		go game.callDoneCmd(game.Name, "game", game.Size)
	}
}

func percent(n, total int64) float32 {
	if total == 0 {
		return float32(0)
	}
	return float32(int(float64(10000)*(float64(n)/float64(total)))) / 100
}

func (t *Game) callDoneCmd(name, tasktype string, size int64) {

	if cmd, env, err := t.e.config.GetCmdConfig(); err == nil {
		cmd := exec.Command(cmd)
		ih := t.InfoHash
		cmd.Env = append(env,
			fmt.Sprintf("CLD_RESTAPI=%s", t.cld.GetStrAttribute("RestAPI")),
			fmt.Sprintf("CLD_PATH=%s", name),
			fmt.Sprintf("CLD_HASH=%s", ih),
			fmt.Sprintf("CLD_TYPE=%s", tasktype),
			fmt.Sprintf("CLD_SIZE=%d", size),
			fmt.Sprintf("CLD_STARTTS=%d", t.StartedAt.Unix()),
			fmt.Sprintf("CLD_FILENUM=%d", len(t.Files)),
		)
		sout, _ := cmd.StdoutPipe()
		serr, _ := cmd.StderrPipe()
		log.Printf("[DoneCmd:%s]%sCMD:`%s' ENV:%s", tasktype, ih, cmd.String(), cmd.Env)
		if err := cmd.Start(); err != nil {
			log.Printf("[DoneCmd:%s]%sERR: %v", tasktype, ih, err)
			return
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go cmdScanLine(sout, &wg, fmt.Sprintf("[DoneCmd:%s]%sO:", log.filteredArg(tasktype, ih)...))
		go cmdScanLine(serr, &wg, fmt.Sprintf("[DoneCmd:%s]%sE:", log.filteredArg(tasktype, ih)...))
		wg.Wait()

		// call Wait will close pipes above
		if err := cmd.Wait(); err != nil {
			log.Printf("[DoneCmd:%s]%sERR: %v", tasktype, ih, err)
			return
		}

		log.Printf("[DoneCmd:%s]%sExit code: %d", tasktype, ih, cmd.ProcessState.ExitCode())
	} else {
		log.Println("[DoneCmd]", t.InfoHash, err)
	}
}
