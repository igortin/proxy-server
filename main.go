package main

import (
	"encoding/json"
	"errors"
	"github.com/darproxy"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	Dir          = "." + strings.Trim(os.Args[0], "./")
	filePath     string
	ConfigPath   = Dir + sep + "config.json"
	PidPath      = Dir + sep + "pid"
	LogPath      = Dir + sep + "proxy.log"
	proxyConfigs = &darproxy.ProxyConfigs{}
	sep          = "/"
	wg           sync.WaitGroup
	VERSION      = "0.0.1"
	isBackground = false
	fd           os.File
	NewLogger    = logrus.New()
)
// Description messages
var (
	ErrNotRunning     = errors.New("process is not running")
	ErrUnableToParse  = errors.New("unable to read and parse process id")
	ErrUnableToFind   = errors.New("unable to find process id")
	ErrUnableToKill   = errors.New("unable to kill process")
	ErrUnableToRemove = errors.New("unable to remove pid file")
	ErrUnableToStart  = errors.New("unable to start procces")

	ErrRunningState   = errors.New("already running or pid file exist")
	)

func main() {
	app := cli.NewApp()
	app.Name = "proxy-cli"
	app.Version = VERSION
	app.Usage = "run --config config.json"
	app.Description = "Proxy service with different politics"
	app.Commands = []cli.Command{
		{
			Name:      "run",
			Usage:     "Run service",
			UsageText: "Run service",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "config, c",
					Usage:       "Load config from `FILE`",
					Destination: &filePath,
				},
				cli.BoolFlag{
					Name:        "daemon, d",
					Usage:       "daemon flag",
					Destination: &isBackground,
				},
			},
			Action: run,
		},
		{
			Name:      "stop",
			Usage:     "Stop service",
			UsageText: "Stop service",
			Action:    stop,
		},
		{
			Name:      "reload",
			Usage:     "Reload service",
			UsageText: "Reload service",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "config, c",
					Usage:       "Load config from `FILE`",
					Destination: &filePath,
				},
			},
			Action: reload,
		},
	}
	sort.Sort(cli.CommandsByName(app.Commands))
	sort.Sort(cli.FlagsByName(app.Flags))
	NewLogger.SetLevel(logrus.DebugLevel)
	fd, _ := os.OpenFile(getLogFilePath(), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0777)
	NewLogger.SetOutput(fd)
	err := app.Run(os.Args)
	if err != nil {
		return
	}

	wg.Wait()
}

func run(ctx *cli.Context) error {
	err := getCfg(proxyConfigs)
	if err != nil {
		NewLogger.Error(ErrUnableToStart," ",err)
		os.Exit(1)
	}
	if isBackground {
		runBackground()
	}
	for _, conf := range proxyConfigs.Configs {
		server := darproxy.NewProxy(
			&http.Server{
				Addr:         conf.Port,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  15 * time.Second,
			}, conf, conf.GraceTimoutStop, NewLogger)
		wg.Add(1)
		go server.Run(&wg)
	}
	return nil
}

func runBackground() error {
	defer os.Exit(0)
	if _, err := os.Stat(getPidFilePath()); err == nil {
		NewLogger.Error(ErrRunningState," ",getPidFilePath())
		os.Exit(1)
		return nil
	}
	if filePath == "" {
		filePath = os.Getenv("HOME") + sep + ConfigPath
	} else {
		filePath, _ = filepath.Abs(filePath)
	}
	cliExec := exec.Command(os.Args[0], "run", "--config", filePath)

	err := cliExec.Start()
	if err != nil {
		NewLogger.Error(err)
		os.Exit(1)
	}
	err = savePID(cliExec.Process.Pid)
	if err != nil {
		NewLogger.Error(err)
		os.Exit(1)
	}
	NewLogger.Debug("background process ID: ", cliExec.Process.Pid, " ", "successfully started")
	return nil
}

func stop(c *cli.Context) error {
	defer os.Exit(0)
	data, err := ioutil.ReadFile(getPidFilePath())
	if err != nil {
		NewLogger.Debug("unable get pid for process: ", err)
	}
	err = clear()
	if err != nil {
		NewLogger.Error("background process ID:", string(data), " ", "could not be stopped: ", err)
		os.Exit(1)
	}
	NewLogger.Debug("background process ID: ", string(data), " ", "successfully stopped")
	return nil
}

func getCfg(serviceCfg *darproxy.ProxyConfigs) error {
	var b []byte

	if filePath == "" {
		filePath = os.Getenv("HOME") + sep + ConfigPath
	} else {
		filePath, _ = filepath.Abs(filePath)
	}
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &serviceCfg)
}

func getPidFilePath() string {
	return os.Getenv("HOME") + sep + PidPath
}

func savePID(pid int) error {
	file, err := os.Create(getPidFilePath())
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write([]byte(strconv.Itoa(pid)))
	if err != nil {
		return err
	}
	return nil
}

func clear() error {
	if _, err := os.Stat(getPidFilePath()); err != nil {
		return ErrNotRunning
	}
	data, err := ioutil.ReadFile(getPidFilePath())
	if err != nil {
		return ErrNotRunning
	}
	ProcessID, err := strconv.Atoi(string(data))
	if err != nil {
		return ErrUnableToParse
	}
	process, err := os.FindProcess(ProcessID)
	if err != nil {
		return ErrUnableToFind
	}
	err = process.Kill()
	if err != nil {
		return ErrUnableToKill
	}
	if os.Remove(getPidFilePath()) != nil {
		return ErrUnableToRemove
	}
	defer fd.Close()
	return nil
}

func reload(c *cli.Context) error {
	err := clear()
	if err != nil {
		return err
	}
	err = runBackground()
	if err != nil {
		return err
	}
	return nil
}


func getLogFilePath() string {
	return os.Getenv("HOME") + sep + LogPath
}