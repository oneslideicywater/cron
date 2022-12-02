package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
)

var DEFAULT_DURATION int = 5
var OPTIONS Options

type Options struct {
	Interval int
	Cmd      []string
}
type Crontab struct {
	ticker    *time.Ticker
	Options   Options
	Worker    Interface
	IsWorking bool
	lock      sync.Mutex
}

func main() {
	stopCh := make(chan struct{}, 1)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	defer close(stopCh)
	defer close(stop)

	app := &cli.App{
		Name:        "cron",
		Usage:       "cron <interval> cmd ...",
		Description: "cron parse the cron syntax and execute cmd periodically",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "duration",
				Aliases:  []string{"d"},
				Usage:    "duration as seconds",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {

			validInterval := DEFAULT_DURATION
			if c.String("duration") != "" {
				parseInt, err := strconv.ParseInt(c.String("duration"), 10, 64)
				if err != nil || parseInt < 0 {
					return errors.New("invalid duration parameter, must be integer and positive")
				}
				validInterval = int(parseInt)
			}
			if c.Args().Len() <= 0 {
				return errors.New("invalid cmd, must not be empty")
			}
			OPTIONS = newOptions(validInterval, c.Args().Slice())
			log.Println(fmt.Sprintf("execute %s at interval %d", OPTIONS.Cmd, OPTIONS.Interval))
			go func() {
				for {
					select {
					case <-stop:
						stopCh <- struct{}{}
						return
					default:
						time.Sleep(time.Second)
					}
				}
			}()
			newCrontab(OPTIONS).Run(stopCh)
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func newOptions(interval int, cmd []string) Options {
	options := Options{
		Interval: interval,
		Cmd:      cmd,
	}

	return options
}

func newCrontab(options Options) *Crontab {
	ret := &Crontab{
		ticker:  time.NewTicker(time.Second * time.Duration(options.Interval)),
		Options: options,
	}
	if runtime.GOOS == "windows" {
		ret.Worker = Windows{}
	} else {
		ret.Worker = Linux{}
	}
	return ret
}

func (c *Crontab) Run(stopCh chan struct{}) {

	for {
		select {
		case <-c.ticker.C:
			fmt.Println(fmt.Sprintf("start execute task %s at %s", c.Options.Cmd, time.Now().String()))
			// if the worker is working in progress, skip
			if c.IsWorking {
				fmt.Println("[INFO] skip due to another worker is working")
				continue
			}
			fmt.Println("starting a new worker")
			// start a new job
			go run(c)
		case <-stopCh:
			fmt.Println("cron exit due to receive stop signal")
			return
		default:
			time.Sleep(time.Second)
		}
	}

}

func run(cron *Crontab) {
	cron.lock.Lock()
	defer cron.lock.Unlock()

	cron.IsWorking = true
	cron.Worker.Run(cron.Options.Cmd)
	cron.IsWorking = false
}

// Interface run specified command with a new go routine
type Interface interface {
	Run(cmd []string)
}

type Windows struct{}

func (Windows) Run(cmd []string) {

	command := exec.Command("cmd", "/c", strings.Join(cmd, " "))
	output, err := command.CombinedOutput()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(Decode(output, GB18030))
}

type Linux struct{}

func (Linux) Run(cmd []string) {
	command := exec.Command("/bin/bash", "-c", strings.Join(cmd, " "))
	output, err := command.CombinedOutput()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("[INFO]task logs:")
	fmt.Println(string(output))
}
