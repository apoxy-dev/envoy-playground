package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	config := os.Args[1]
	command := os.Args[2]
	result, err := run(config, command)
	resp := RunResponse{
		Result: result,
		Error:  toString(err),
	}
	jsonStr, _ := json.Marshal(resp)
	fmt.Println(string(jsonStr))
}

func toString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

type RunResponse struct {
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func run(envoyConfig string, command string) (string, error) {
	configFile, err := ioutil.TempFile("/tmp", "envoy.*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp config file: %s", err)
	}
	defer os.Remove(configFile.Name())
	if _, err := configFile.Write([]byte(envoyConfig)); err != nil {
		return "", fmt.Errorf("failed to write config file: %s", err)
	}

	httpbin_cmd := exec.Command("go-httpbin", "-port", "7777")
	if err := httpbin_cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start go-httpbin: %s", err)
	}
	defer kill(httpbin_cmd)

	envoy_cmd := exec.Command("envoy", "-c", configFile.Name(), "--log-level", "error")
	envoy_cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	var stderr bytes.Buffer
	envoy_cmd.Stderr = &stderr
	ch := make(chan error)
	go func() {
		ch <- envoy_cmd.Run()
	}()

	// Check for errors
	select {
	case err := <-ch:
		if err == nil {
			break
		}
		return "", fmt.Errorf("envoy failed to start (exec error: %v). stderr:\n\n %b", err, stderr)
	case <-time.After(100 * time.Millisecond):
		defer term(envoy_cmd)
		break
	}
	curlArgs := strings.Split(strings.TrimSpace(command), " ")

	if curlArgs[0] != "curl" && curlArgs[0] != "http" {
		return "", fmt.Errorf("command must start with 'curl' or 'http'")
	}

	output, err := exec.Command(curlArgs[0], curlArgs[1:]...).CombinedOutput()
	if err != nil {
		return string(output), err
	}

	if stderr.Len() > 0 {
		err := fmt.Errorf("envoy error logs:\n\n %s", stderr.String())
		return string(output), err
	}
	return string(output), nil
}

func kill(cmd *exec.Cmd) {
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
}

func term(cmd *exec.Cmd) {
	if cmd.Process != nil {
		cmd.Process.Signal(syscall.SIGTERM)
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
