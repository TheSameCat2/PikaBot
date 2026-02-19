package dockerctl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
)

type Controller struct {
	cli           *client.Client
	containerName string
}

type Status struct {
	Exists  bool
	Running bool
	State   string
}

func New(containerName string) (*Controller, error) {
	if strings.TrimSpace(containerName) == "" {
		return nil, errors.New("container name is required")
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	return &Controller{cli: cli, containerName: containerName}, nil
}

func (c *Controller) Close() error {
	return c.cli.Close()
}

func (c *Controller) Status(ctx context.Context) (Status, error) {
	inspect, err := c.cli.ContainerInspect(ctx, c.containerName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return Status{Exists: false}, nil
		}
		return Status{}, fmt.Errorf("inspect container %q: %w", c.containerName, err)
	}

	state := ""
	running := false
	if inspect.ContainerJSONBase != nil && inspect.ContainerJSONBase.State != nil {
		state = inspect.ContainerJSONBase.State.Status
		running = inspect.ContainerJSONBase.State.Running
	}

	return Status{Exists: true, Running: running, State: state}, nil
}

func (c *Controller) Start(ctx context.Context) error {
	if err := c.cli.ContainerStart(ctx, c.containerName, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container %q: %w", c.containerName, err)
	}
	return nil
}

func (c *Controller) Stop(ctx context.Context, timeout time.Duration) error {
	seconds := int(timeout.Seconds())
	if seconds < 1 {
		seconds = 10
	}
	if err := c.cli.ContainerStop(ctx, c.containerName, container.StopOptions{Timeout: &seconds}); err != nil {
		return fmt.Errorf("stop container %q: %w", c.containerName, err)
	}
	return nil
}
