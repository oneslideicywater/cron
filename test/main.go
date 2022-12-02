package main

import (
	"os"
	"strconv"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		return
	}
	interval := os.Args[1]
	if s, err := strconv.ParseInt(interval, 10, 64); err == nil {
		time.Sleep(time.Duration(s) * time.Second)
	}
	return
}
