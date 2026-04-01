package ap

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Sandbox interface {
	Setup(id string, forgeRepo string, forgeToken string) error
	Exec(id string, command string, timeout time.Duration) (string, error)
	Read(id string, path string) ([]byte, error)
	Write(id string, path string, data []byte) error
	URL(id string, port int) (string, error)
}

type SandboxDocker struct {
	image string
}

func NewSandboxDocker(image string) Sandbox {
	return &SandboxDocker{image: Or(image, "ubuntu:24.04")}
}

func (s *SandboxDocker) Setup(id string, forgeRepo string, forgeToken string) error {
	name := containerName(id)
	if out, err := exec.Command("docker", "run", "-d", "--name", name, s.image, "sleep", "infinity").CombinedOutput(); err != nil {
		return fmt.Errorf("docker run: %w: %s", err, bytes.TrimSpace(out))
	}
	if out, err := exec.Command("docker", "exec", name, "sh", "-c", "command -v git || (apt-get update -qq && apt-get install -y -qq git)").CombinedOutput(); err != nil {
		return fmt.Errorf("docker exec git install: %w: %s", err, bytes.TrimSpace(out))
	}
	host := forgeHost(forgeRepo)
	cmd := exec.Command("docker", "exec", "-i", name, "sh", "-c", "cat > /root/.netrc")
	cmd.Stdin = strings.NewReader(fmt.Sprintf("machine %s login x-access-token password %s\n", host, forgeToken))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker exec netrc: %w: %s", err, bytes.TrimSpace(out))
	}
	if out, err := exec.Command("docker", "exec", name, "chmod", "600", "/root/.netrc").CombinedOutput(); err != nil {
		return fmt.Errorf("docker exec chmod: %w: %s", err, bytes.TrimSpace(out))
	}
	if _, err := exec.Command("docker", "exec", name, "git", "clone", strings.TrimSpace(forgeRepo), "/work").CombinedOutput(); err != nil {
		return fmt.Errorf("docker exec git clone: %w", err)
	}
	return nil
}

func (s *SandboxDocker) Exec(id string, command string, timeout time.Duration) (string, error) {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()
	out, err := exec.CommandContext(ctx, "docker", "exec", containerName(id), "sh", "-c", "cd /work && "+command).CombinedOutput()
	return string(out), err
}

func (s *SandboxDocker) Read(id string, path string) ([]byte, error) {
	return exec.Command("docker", "exec", containerName(id), "cat", path).Output()
}

func (s *SandboxDocker) Write(id string, path string, data []byte) error {
	if out, err := exec.Command("docker", "exec", containerName(id), "mkdir", "-p", filepath.Dir(path)).CombinedOutput(); err != nil {
		return fmt.Errorf("docker exec mkdir: %w: %s", err, bytes.TrimSpace(out))
	}
	cmd := exec.Command("docker", "exec", "-i", containerName(id), "sh", "-c", `cat > "$1"`, "--", path)
	cmd.Stdin = bytes.NewReader(data)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker exec write: %w: %s", err, bytes.TrimSpace(out))
	}
	return nil
}

func (s *SandboxDocker) URL(id string, port int) (string, error) {
	out, err := exec.Command("docker", "inspect", "-f", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", containerName(id)).Output()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%s:%d", strings.TrimSpace(string(out)), port), nil
}

func containerName(id string) string {
	return "sandbox-" + id
}

func forgeHost(forgeRepo string) string {
	forgeRepo = strings.TrimSpace(forgeRepo)
	for _, prefix := range []string{"https://", "http://"} {
		if rest, ok := strings.CutPrefix(forgeRepo, prefix); ok {
			forgeRepo = rest
			break
		}
	}
	host, _, _ := strings.Cut(forgeRepo, "/")
	return host
}
