package server

var hasEmailServiceFailed = false

func StartServer() error {
	apiChannel := make(chan error)
	go func() {
		apiChannel <- serveApi()
	}()
	emailChannel := make(chan error)
	go func() {
		emailChannel <- serveEmail()
	}()

	isRunning := true
	for isRunning {
		select {
		case err := <-apiChannel:
			if err != nil {
				go func() {
					// Restart the server if it crashes
					apiChannel <- serveApi()
				}()
			}
		case err := <-emailChannel:
			if err != nil {
				hasEmailServiceFailed = true
			}
		}
	}
	return nil
}
