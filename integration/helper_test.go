//go:build integration

package integration_test

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
)

const (
	port = 8080
)

var geckoDriverPath string

func init() {
	_, f, _, _ := runtime.Caller(0)
	geckoDriverPath = filepath.Join(filepath.Dir(f), "vendor", "geckodriver")
}

func getRemote(t *testing.T) selenium.WebDriver {
	// Start Xvfb for headless browser testing.
	fbuf, err := selenium.NewFrameBuffer()
	require.NoError(t, err)
	t.Cleanup(func() {
		fbuf.Stop()
	})

	// Start geckodriver bound to 127.0.0.1 (the default).
	cmd := exec.Command(geckoDriverPath, "--port", strconv.Itoa(port))
	cmd.Env = append(os.Environ(), "DISPLAY=:"+fbuf.Display, "XAUTHORITY="+fbuf.AuthPath)
	err = cmd.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})

	// Wait for geckodriver to be ready.
	addr := fmt.Sprintf("http://127.0.0.1:%d", port)
	for i := 0; i < 30; i++ {
		time.Sleep(time.Second)
		resp, err := http.Get(addr + "/status")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
	}

	// Connect to the WebDriver instance running locally.
	caps := selenium.Capabilities{"browserName": "firefox"}
	wd, err := selenium.NewRemote(caps, addr)
	require.NoError(t, err)
	t.Cleanup(func() {
		wd.Quit()
	})

	return wd
}
