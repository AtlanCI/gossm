package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os/exec"
	"runtime"
	"time"

	"github.com/AtlanCI/gossm"
	"github.com/AtlanCI/gossm/conf"
	"github.com/AtlanCI/gossm/logger"
)

var configPath = flag.String("config", "configs/default.json", "configuration file")
var logPath = flag.String("log", "logs/from-"+time.Now().Format("2006-01-02")+".log", "log file")
var address = flag.String("http", ":8080", "address for http server")
var nolog = flag.Bool("nolog", false, "disable logging to file only")
var logfilter = flag.String("logfilter", "", "text to filter log by (both console and file)")

func main() {
	flag.Parse()
	jsonData, err := ioutil.ReadFile(*configPath)
	if err != nil {
		panic("error reading from configuration file")
	}

	if *nolog == true {
		logger.Disable()
	}

	if *logfilter != "" {
		logger.Filter(*logfilter)
	}

	logger.SetFilename(*logPath)

	config := conf.NewConfig(jsonData)
	monitor := gossm.NewMonitor(config)
	go gossm.RunHttp(*address, monitor)
	Open("http://127.0.0.1" + *address)
	monitor.Run()
}

// 不同平台启动指令不同
var commands = map[string]string{
	"windows": "explorer",
	"darwin":  "open",
	"linux":   "xdg-open",
}

func Open(uri string) error {
	// runtime.GOOS获取当前平台
	run, ok := commands[runtime.GOOS]
	if !ok {
		return fmt.Errorf("don't know how to open things on %s platform", runtime.GOOS)
	}

	cmd := exec.Command(run, uri)
	return cmd.Run()
}
