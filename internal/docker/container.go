package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type Container struct {
	ctx      context.Context
	dc       *client.Client
	event    EventHandler
	ID       string
	Name     string
	Veth     string
	VethPeer string
	TcRate   string
	TcCeil   string
}

func NewContainer(ctx context.Context, dc *client.Client) *Container {
	c := &Container{ctx: ctx, dc: dc}
	go c.eventWatch()
	return c
}

// build containers from container id
func (c *Container) buildContainers(containerID string, labels map[string]string) ([]*Container, error) {
	name, err := c.getName(containerID)
	if err != nil {
		return nil, fmt.Errorf("getName error: %v", err)
	}
	sandboxKey, err := c.getSandboxKey(containerID)
	if err != nil {
		return nil, fmt.Errorf("getSandboxKey error: %v", err)
	}
	veths, err := c.GetVeths(name, sandboxKey)
	if err != nil {
		return nil, fmt.Errorf("GetVeths error: %v", err)
	}
	rate, ceil := c.getLabelTC(labels)

	var containers []*Container
	for _, vethPair := range veths {
		containers = append(containers, &Container{
			ID:       containerID[:12],
			Name:     name,
			Veth:     vethPair[0],
			VethPeer: vethPair[1],
			TcRate:   rate,
			TcCeil:   ceil,
		})
	}
	return containers, nil
}

func (c *Container) GetRunningList() ([]*Container, error) {
	f := filters.NewArgs()
	f.Add("label", "org.label-schema.tc.enabled=1")
	f.Add("status", "running")
	containerList, err := c.dc.ContainerList(c.ctx, types.ContainerListOptions{Filters: f})
	if err != nil {
		return nil, fmt.Errorf("ContainerList error: %v", err)
	}

	var containers []*Container
	for _, container := range containerList {
		c, err := c.buildContainers(container.ID, container.Labels)
		if err != nil {
			return nil, err
		}

		// append c to containers
		containers = append(containers, c...)
	}
	return containers, nil
}

func (c *Container) getName(containerID string) (string, error) {
	cJson, err := c.dc.ContainerInspect(c.ctx, containerID)
	if err != nil {
		return "", err
	}
	return strings.TrimLeft(cJson.Name, "/"), nil
}

func (c *Container) getSandboxKey(containerID string) (string, error) {
	cJson, err := c.dc.ContainerInspect(c.ctx, containerID)
	if err != nil {
		return "", err
	}
	return cJson.NetworkSettings.SandboxKey, nil
}

func (c *Container) getLabelTC(labels map[string]string) (string, string) {
	rate, hasRate := labels["org.label-schema.tc.rate"]
	ceil, hasCeil := labels["org.label-schema.tc.ceil"]
	if !hasRate && !hasCeil {
		return "10000mbps", "10000mbps"
	}
	if !hasRate {
		rate = ceil
	}
	if !hasCeil {
		ceil = rate
	}
	return rate, ceil
}
