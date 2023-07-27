package main

import (
	"encoding/json"
	"fmt"
	"io"
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
	httpbin_cmd := exec.Command("go-httpbin", "-port", "7777")
	if err := httpbin_cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start go-httpbin: %s", err)
	}
	defer kill(httpbin_cmd)

	envoy_cmd := exec.Command("func-e", "run", "--config-yaml", envoyConfig, "--log-level", "error")
	envoy_cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	ch := make(chan error)
	stderr, err := envoy_cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stderr pipe: %s", err)
	}
	go func() {
		ch <- envoy_cmd.Run()
	}()

	// Check for errors
	select {
	case err := <-ch:
		if err == nil {
			break
		}
		fmt.Println("Envoy failed to start: ", err)
		logs, _ := io.ReadAll(stderr)
		return "", fmt.Errorf("envoy failed to start. Error logs:\n\n %s", string(logs))
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

	logs, _ := io.ReadAll(stderr)
	if logs != nil {
		err := fmt.Errorf("envoy error logs:\n\n %s", string(logs))
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
