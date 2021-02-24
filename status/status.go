package status

import (
	"context"
	"fmt"
	"time"

	"github.com/xybydy/gdutils/counter"
)

const (
	StatusReadPath = iota
	StatusCopy
	StatusCreateFolder
)

func PrintStatus(ctx context.Context, pending *counter.Counter, done *counter.Counter, statusType int) {
	var printText string

	switch statusType {
	case StatusReadPath:
		printText = "%s | Read %d | Pending %d |"
	case StatusCopy:
		printText = "%s | Files Copied: %d | Files Pending: %d |"
	case StatusCreateFolder:
		printText = "%s | Folders Created %d | Folders Pending %d |"
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			pending.Stop()
			now := time.Now().Format("15:04:05")
			message := fmt.Sprintf(printText, now, done.Get(), pending.Get())
			printProgress(message)
			return
		case <-ticker.C:
			now := time.Now().Format("15:04:05")
			message := fmt.Sprintf(printText, now, done.Get(), pending.Get())
			printProgress(message)
		}
	}
}

func printProgress(msg string) {
	fmt.Printf("\r\033[K%s", msg)
}
