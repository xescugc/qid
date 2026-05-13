package integration_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
)

const (
	// These paths will be different on your system.
	seleniumPath    = "/home/xescugc/repos/qid/integration/vendor/selenium-server.jar"
	geckoDriverPath = "/home/xescugc/repos/qid/integration/vendor/geckodriver"
	port            = 8080
)

func getRemote(t *testing.T) selenium.WebDriver {
	opts := []selenium.ServiceOption{
		selenium.StartFrameBuffer(),           // Start an X frame buffer for the browser to run in.
		selenium.GeckoDriver(geckoDriverPath), // Specify the path to GeckoDriver in order to use Firefox.
		//selenium.Output(os.Stderr),            // Output debug information to STDERR.
	}
	//selenium.SetDebug(true)
	service, err := selenium.NewSeleniumService(seleniumPath, port, opts...)
	require.NoError(t, err)
	t.Cleanup(func() {
		service.Stop()
	})

	// Connect to the WebDriver instance running locally.
	caps := selenium.Capabilities{"browserName": "firefox"}
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
	require.NoError(t, err)
	t.Cleanup(func() {
		wd.Quit()
	})

	return wd
}
