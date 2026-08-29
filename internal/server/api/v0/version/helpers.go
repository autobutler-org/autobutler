package v0_version

import (
	"fmt"
	"os"
	"time"
)

func restartQuark() {
	fmt.Println("Update complete. Exiting to allow process manager (launchctl/systemd) to restart...")
	time.Sleep(time.Second * 2)
	os.Exit(0)
}
