package collector

import (
	"fmt"
	"os"
	"os/user"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

const userHZ = 100

type (
	groupKey struct {
		account   string
		groupname string
	}

	procGroup struct {
		name            string
		account         string
		cpuSystem       float64
		cpuUser         float64
		memVirt         uint64
		memRss          uint64
		numProcs        uint64
		numThreads      uint64
		oldestStartTime float64
	}

	procCollector struct {
		procfsPath      string
		matchnamer      MatchNamer
		collectFn       func(chan<- prometheus.Metric)
		scrapeErrors    *prometheus.Desc
		cpu             *prometheus.Desc
		memory          *prometheus.Desc
		numProcs        *prometheus.Desc
		numThreads      *prometheus.Desc
		oldestStartTime *prometheus.Desc
		errors          struct {
			scrape int
		}
	}
)

func NewProcCollector(procfsPath string, matchnamer MatchNamer) prometheus.Collector {
	ns := "proc_"

	return &procCollector{
		procfsPath: procfsPath,
		matchnamer: matchnamer,

		scrapeErrors: prometheus.NewDesc(
			ns+"scrape_errors",
			"Error collecting proc metrics",
			nil,
			nil,
		),
		cpu: prometheus.NewDesc(
			ns+"cpu_seconds_total",
			"Total user CPU time spent in seconds.",
			[]string{"account", "groupname", "mode"},
			nil,
		),
		memory: prometheus.NewDesc(
			ns+"memory_bytes",
			"Used amount of memory in bytes.",
			[]string{"account", "groupname", "memtype"},
			nil,
		),
		numProcs: prometheus.NewDesc(
			ns+"num_procs",
			"Number of processes.",
			[]string{"account", "groupname"},
			nil,
		),
		numThreads: prometheus.NewDesc(
			ns+"num_threads",
			"Number of threads.",
			[]string{"account", "groupname"},
			nil,
		),
		oldestStartTime: prometheus.NewDesc(
			ns+"oldest_start_time_seconds",
			"Oldest process start time in seconds.",
			[]string{"account", "groupname"},
			nil,
		),
	}
}

// Describe returns all descriptions of the collector.
func (c *procCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.scrapeErrors
	ch <- c.cpu
	ch <- c.memory
	ch <- c.numProcs
	ch <- c.numThreads
	ch <- c.oldestStartTime
}

// Collect returns the current state of all metrics of the collector.
func (c *procCollector) Collect(ch chan<- prometheus.Metric) {
	procGroups, _ := c.readProcGroups()

	for _, g := range procGroups {
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, g.cpuSystem, g.account, g.name, "system")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, g.cpuUser, g.account, g.name, "user")
		ch <- prometheus.MustNewConstMetric(c.memory, prometheus.GaugeValue, float64(g.memVirt), g.account, g.name, "virtual")
		ch <- prometheus.MustNewConstMetric(c.memory, prometheus.GaugeValue, float64(g.memRss), g.account, g.name, "resident")
		ch <- prometheus.MustNewConstMetric(c.numProcs, prometheus.GaugeValue, float64(g.numProcs), g.account, g.name)
		ch <- prometheus.MustNewConstMetric(c.numThreads, prometheus.GaugeValue, float64(g.numThreads), g.account, g.name)
		ch <- prometheus.MustNewConstMetric(c.oldestStartTime, prometheus.GaugeValue, float64(g.oldestStartTime), g.account, g.name)
	}

	ch <- prometheus.MustNewConstMetric(c.scrapeErrors, prometheus.CounterValue, float64(c.errors.scrape))
}

func (c *procCollector) readProcGroups() (map[groupKey]*procGroup, error) {
	// list processes
	procs, err := procfs.AllProcs()
	if err != nil {
		c.errors.scrape += 1
		return nil, err
	}

	fstat, err := procfs.NewStat()
	if err != nil {
		c.errors.scrape += 1
	}

	var (
		bootTime   = uint64(fstat.BootTime)
		procGroups = make(map[groupKey]*procGroup, 100)
	)

	for _, p := range procs {
		// read comm & cmdline
		stat, err := p.NewStat()
		if err != nil {
			c.errors.scrape += 1
			continue
		}
		cmdline, err := p.CmdLine()
		if err != nil {
			c.errors.scrape += 1
			continue
		}

		// match
		comm := stat.Comm
		nacl := NameAndCmdline{Name: comm, Cmdline: cmdline}
		wanted, gname := c.matchnamer.MatchAndName(nacl)

		if !wanted {
			continue
		}

		// read metrics
		account, err := getProcAccount(p.PID)
		if err != nil {
			c.errors.scrape += 1
		}
		cpuSystem := float64(stat.STime) / userHZ
		cpuUser := float64(stat.UTime) / userHZ
		memVirt := uint64(stat.VirtualMemory())
		memRss := uint64(stat.ResidentMemory())
		numThreads := uint64(stat.NumThreads)
		startTime := float64(bootTime) + (float64(stat.Starttime) / userHZ)

		// get a group
		gkey := groupKey{account, gname}
		g := procGroups[gkey]

		if g == nil {
			g = &procGroup{name: gname, account: account}
			procGroups[gkey] = g
		}

		// update group
		g.cpuSystem += cpuSystem
		g.cpuUser += cpuUser
		g.memVirt += memVirt
		g.memRss += memRss
		g.numProcs += 1
		g.numThreads += numThreads
		if g.oldestStartTime == 0 || startTime < g.oldestStartTime {
			g.oldestStartTime = startTime
		}
	}

	return procGroups, nil
}

func getProcAccount(pid int) (string, error) {
	fi, err := os.Stat(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		fmt.Println(fmt.Errorf("Stat error for %d: %v", pid, err))
		return "", err
	}

	fstat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		fmt.Println(fmt.Errorf("Stat_t is not available for %d: %v", pid, err))
		return "", err
	}

	account, err := user.LookupId(fmt.Sprint(fstat.Uid))
	if err != nil {
		fmt.Println(fmt.Errorf("User lookup error for %d: %v", pid, err))
		return "", err
	}

	return account.Username, nil
}
