package health

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"sync"
	"time"

	"cosmossdk.io/errors"
	"github.com/google/pprof/profile"
	"github.com/pelletier/go-toml/v2"

	"github.com/allora-network/allora-chain/log"
	"github.com/allora-network/allora-chain/utils"
)

type Nurse struct {
	cfg      NurseConfig
	checks   map[string]CheckFunc
	checksMu sync.RWMutex
	chGather chan gatherRequest

	log.Logger

	chStop chan struct{}
	wgDone sync.WaitGroup
}

type NurseConfig struct {
	ProfileRoot    string
	PollInterval   time.Duration
	GatherDuration time.Duration
	MaxProfileSize utils.FileSize

	CPUProfileRate       int
	MemProfileRate       int
	BlockProfileRate     int
	MutexProfileFraction int

	MemThreshold       utils.FileSize
	GoroutineThreshold int

	Logger log.Logger
}

func MustReadConfigTOML(path string) NurseConfig {
	cfg, err := ReadConfigTOML(path)
	if err != nil {
		panic(err)
	}
	return cfg
}

func ReadConfigTOML(path string) (NurseConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return NurseConfig{}, errors.Wrap(err, "while opening nurse toml")
	}
	defer file.Close()

	var cfg NurseConfig
	err = toml.NewDecoder(file).Decode(&cfg)
	if err != nil {
		return NurseConfig{}, errors.Wrap(err, "while decoding nurse toml")
	}
	return cfg, nil
}

type CheckFunc func() (unwell bool, meta Meta)

type gatherRequest struct {
	reason string
	meta   Meta
}

type Meta map[string]any

const profilePerms = 0666

func NewNurse(cfg NurseConfig) *Nurse {
	return &Nurse{
		cfg:      cfg,
		Logger:   cfg.Logger,
		checks:   make(map[string]CheckFunc),
		checksMu: sync.RWMutex{},
		chGather: make(chan gatherRequest, 1),
		chStop:   make(chan struct{}),
		wgDone:   sync.WaitGroup{},
	}
}

func (n *Nurse) Start() error {
	// This must be set *once*, and it must occur as early as possible
	runtime.MemProfileRate = n.cfg.MemProfileRate

	err := utils.EnsureDirAndMaxPerms(n.cfg.ProfileRoot, 0700)
	if err != nil {
		return err
	}

	n.AddCheck("mem", n.checkMem)
	n.AddCheck("goroutines", n.checkGoroutines)

	n.wgDone.Add(1)
	go func() {
		defer n.wgDone.Done()

		for {
			select {
			case <-n.chStop:
				return
			case <-time.After(n.cfg.PollInterval):
			}

			func() {
				n.checksMu.RLock()
				defer n.checksMu.RUnlock()
				for reason, checkFunc := range n.checks {
					if unwell, meta := checkFunc(); unwell {
						n.GatherVitals(reason, meta)
						break
					}
				}
			}()
		}
	}()

	n.wgDone.Add(1)
	go func() {
		defer n.wgDone.Done()

		for {
			select {
			case <-n.chStop:
				return
			case req := <-n.chGather:
				n.gatherVitals(req.reason, req.meta)
			}
		}
	}()

	return nil
}

func (n *Nurse) Close() error {
	close(n.chStop)
	n.wgDone.Wait()
	return nil
}

func (n *Nurse) AddCheck(reason string, checkFunc CheckFunc) {
	n.checksMu.Lock()
	defer n.checksMu.Unlock()
	n.checks[reason] = checkFunc
}

func (n *Nurse) GatherVitals(reason string, meta Meta) {
	select {
	case n.chGather <- gatherRequest{reason, meta}:
	default:
	}
}

func (n *Nurse) checkMem() (bool, Meta) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	unwell := memStats.Alloc >= uint64(n.cfg.MemThreshold)
	if !unwell {
		return false, nil
	}
	return true, Meta{
		"mem_alloc": utils.FileSize(memStats.Alloc),
		"threshold": n.cfg.MemThreshold,
	}
}

func (n *Nurse) checkGoroutines() (bool, Meta) {
	num := runtime.NumGoroutine()
	unwell := num >= n.cfg.GoroutineThreshold
	if !unwell {
		return false, nil
	}
	return true, Meta{
		"num_goroutines": num,
		"threshold":      n.cfg.GoroutineThreshold,
	}
}

func (n *Nurse) gatherVitals(reason string, meta Meta) {
	loggerFields := log.Fields{"reason": reason}.Merge(log.Fields(meta))

	n.Debug("nurse is gathering vitals", loggerFields)

	size, err := n.totalProfileBytes()
	if err != nil {
		n.Error("could not fetch total profile bytes", loggerFields.With("error", err).Slice()...)
		return
	} else if size >= uint64(n.cfg.MaxProfileSize) {
		n.Warn("cannot write pprof profile, total profile size exceeds configured MaxProfileSize",
			loggerFields.With("total", size, "max", n.cfg.MaxProfileSize).Slice()...,
		)
		return
	}

	runtime.SetCPUProfileRate(n.cfg.CPUProfileRate)
	defer runtime.SetCPUProfileRate(0)
	runtime.SetBlockProfileRate(n.cfg.BlockProfileRate)
	defer runtime.SetBlockProfileRate(0)
	runtime.SetMutexProfileFraction(n.cfg.MutexProfileFraction)
	defer runtime.SetMutexProfileFraction(0)

	now := time.Now()

	var wg sync.WaitGroup
	wg.Add(9)

	go n.appendLog(now, reason, meta, &wg)
	go n.gatherCPU(now, &wg)
	go n.gatherTrace(now, &wg)
	go n.gather("allocs", now, &wg)
	go n.gather("block", now, &wg)
	go n.gather("goroutine", now, &wg)
	go n.gather("heap", now, &wg)
	go n.gather("mutex", now, &wg)
	go n.gather("threadcreate", now, &wg)

	// Because you can't `select` on a `sync.WaitGroup`, we present the following Lovecraftian horror
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		wg.Wait()
	}()

	select {
	case <-n.chStop:
	case <-ch:
	}
}

func (n *Nurse) appendLog(now time.Time, reason string, meta Meta, wg *sync.WaitGroup) {
	defer wg.Done()

	filename := filepath.Join(n.cfg.ProfileRoot, "nurse.log")
	mode := os.O_APPEND | os.O_CREATE | os.O_WRONLY

	file, err := os.OpenFile(filename, mode, profilePerms)
	if err != nil {
		n.Error("could not append to log", "error", err)
		return
	}
	defer file.Close()

	lines := make([]string, 2+len(meta))
	lines[0] = fmt.Sprintf("==== %v", now)
	lines[1] = fmt.Sprintf("reason: %v", reason)
	i := 0
	for k, v := range meta {
		lines[i] = fmt.Sprintf("- %v: %v", k, v)
		i++
	}
	entry := strings.Join(lines, "\n")

	_, err = file.Write([]byte(entry + "\n"))
	if err != nil {
		n.Error("could not append to log", "error", err)
		return
	}
}

func (n *Nurse) gatherCPU(now time.Time, wg *sync.WaitGroup) {
	defer wg.Done()

	file, err := n.openFile(now, "cpu")
	if err != nil {
		n.Error("could not write cpu profile", "error", err)
		return
	}
	defer file.Close()

	err = pprof.StartCPUProfile(file)
	if err != nil {
		n.Error("could not start cpu profile", "error", err)
		return
	}
	defer pprof.StopCPUProfile()

	select {
	case <-n.chStop:
	case <-time.After(n.cfg.GatherDuration):
	}
}

func (n *Nurse) gatherTrace(now time.Time, wg *sync.WaitGroup) {
	defer wg.Done()

	file, err := n.openFile(now, "trace")
	if err != nil {
		n.Error("could not write trace profile", "error", err)
		return
	}
	defer file.Close()

	err = trace.Start(file)
	if err != nil {
		n.Error("could not start trace profile", "error", err)
		return
	}
	defer trace.Stop()

	select {
	case <-n.chStop:
	case <-time.After(n.cfg.GatherDuration):
	}
}

func (n *Nurse) gather(typ string, now time.Time, wg *sync.WaitGroup) {
	defer wg.Done()

	p := pprof.Lookup(typ)
	if p == nil {
		n.Error("invariant violation: pprof type does not exist", "type", typ)
		return
	}

	p0, err := collectProfile(p)
	if err != nil {
		n.Error("could not collect profile", "type", typ, "error", err)
		return
	}

	t := time.NewTimer(n.cfg.GatherDuration)
	defer t.Stop()

	select {
	case <-n.chStop:
		return
	case <-t.C:
	}

	p1, err := collectProfile(p)
	if err != nil {
		n.Error("could not collect profile", "type", typ, "error", err)
		return
	}
	ts := p1.TimeNanos
	dur := p1.TimeNanos - p0.TimeNanos

	p0.Scale(-1)

	p1, err = profile.Merge([]*profile.Profile{p0, p1})
	if err != nil {
		n.Error("could not compute delta for profile", "type", typ, "error", err)
		return
	}

	p1.TimeNanos = ts // set since we don't know what profile.Merge set for TimeNanos.
	p1.DurationNanos = dur

	file, err := n.openFile(now, typ)
	if err != nil {
		n.Error("could not write profile", "type", typ, "error", err)
		return
	}
	defer file.Close()

	err = p1.Write(file)
	if err != nil {
		n.Error("could not write profile", "type", typ, "error", err)
		return
	}
}

func collectProfile(p *pprof.Profile) (*profile.Profile, error) {
	var buf bytes.Buffer
	if err := p.WriteTo(&buf, 0); err != nil {
		return nil, err
	}
	ts := time.Now().UnixNano()
	p0, err := profile.Parse(&buf)
	if err != nil {
		return nil, err
	}
	p0.TimeNanos = ts
	return p0, nil
}

func (n *Nurse) openFile(now time.Time, typ string) (*os.File, error) {
	filename := filepath.Join(n.cfg.ProfileRoot, fmt.Sprintf("%v.%v.pprof", now, typ))
	mode := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	return os.OpenFile(filename, mode, profilePerms)
}

func (n *Nurse) totalProfileBytes() (uint64, error) {
	entries, err := os.ReadDir(n.cfg.ProfileRoot)
	if os.IsNotExist(err) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	var size uint64
	for _, entry := range entries {
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".pprof" && entry.Name() != "nurse.log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return 0, err
		}
		size += uint64(info.Size())
	}
	return size, nil
}
