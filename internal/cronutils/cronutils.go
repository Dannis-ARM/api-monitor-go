package cronutils

import (
	"log"
	"os/exec"
	"time"
)

// InitCronJob initializes and starts a cron job that executes "ls -al" every 6 hours.
func InitCronJob() {
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()

		// Execute immediately at startup
		executeLsAl()

		for range ticker.C {
			executeLsAl()
		}
	}()
	log.Println("Cron job for 'ls -al' started, running every 6 hours.")
}

func executeLsAl() {
	cmd := exec.Command("cmd", "/c", "dir", "/a")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error executing 'ls -al': %v\nOutput: %s", err, output)
		return
	}
	log.Printf("Cron job 'ls -al' executed. Output:\n%s", output)
}
