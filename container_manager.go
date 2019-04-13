package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func uint64P(u uint64) *uint64 {
	p := u
	return &p
}

// intP returns a pointer to an int
func int64P(i int64) *int64 {
	p := i
	return &p
}

// boolP returns a pointer to a bool
func boolP(b bool) *bool {
	p := b
	return &p
}

## Get Stats from the runnning containers
func GetStats(a *ManagedContainer, c *client.Client) (*types.Stats, error) {
	cs, err := c.ContainerStats(context.Background(), a.Name, false)
	if err != nil {
		return nil, err
	}
	defer cs.Body.Close()

	d := json.NewDecoder(cs.Body)
	d.UseNumber()

	var stats types.Stats
	if err := d.Decode(&stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// ContainerInspect returns the information which can decide if the container is current;y running
// or not.
func ContainerInspect(a *ManagedContainer, c *client.Client) (*types.ContainerJSON, error) {
	containerJSON, err := c.ContainerInspect(context.Background(), a.Name)
	if err != nil {
		return nil, err
	}

	return &containerJSON, nil
}

// InitCheckers returns a slice of containers with all the info needed to run a
// check on the container. 
func InitCheckers(c *Conf) []ManagedContainer {
	// Taking the values from the conf and adding them into the ManagedContainers
	var containers []ManagedContainer
	for _, v := range c.Containers {
		containers = append(containers, ManagedContainer{
			Name: v.Name,
			Action: &Action{
				Messages: []error{},
			},
			CPUCheck: &MetricCheck{
				Limit:       v.MaxCPU,
				ActionActive: false,
			},
			MemCheck: &MetricCheck{
				Limit:       v.MaxMem,
				ActionActive: false,
			},
			PIDCheck: &MetricCheck{
				Limit:       v.MinProcs,
				ActionActive: false,
			},
			ExistenceCheck: &StaticCheck{
				Expected:    boolP(true),
				ActionActive: false,
			},
			RunningCheck: &StaticCheck{
				Expected:    v.ExpectedRunning,
				ActionActive: false,
			},
		})
	}
	return containers
}

// CheckContainers goes through and checks all the containers in a loop
func CheckContainers(cnt []ManagedContainer, cli *client.Client, a *Action) {
	for _, c := range cnt {
		// make sure we have a clean Action for this loop
		c.Action.Clear()

		// handling whether the container exists, if these checks fail, the checking
		// process should stop
		j, err := ContainerInspect(&c, cli)
		c.CheckStatics(j, err)

		// if an action should be executed that means it either failed or running
		// checks which means that nothing more can be checked
		if c.ChecksShouldStop() {
			a.Concat(c.Action) // add the action in the host
			continue
		}

		s, err := GetStats(&c, cli)
		c.CheckMetrics(s, err)

		if c.Action.ShouldSend() {
			a.Concat(c.Action)
		}
	}
}

// Monitor containers contains all the calls for the main 
func Monitor(c *Conf, a *Action) {
	cli, err := client.NewEnvClient()
	if err != nil {
		log.Fatal(err)
	}

	cnt := InitCheckers(c)

	switch c.Iterations {
	case 0:
		for {
			a.Clear()
			CheckContainers(cnt, cli, a)
			a.Evaluate()
			time.Sleep(time.Duration(c.Duration) * time.Millisecond)
		}
	default:
		for i := uint64(0); i < c.Iterations; i++ {
			a.Clear()
			CheckContainers(cnt, cli, a)
			a.Evaluate()
			time.Sleep(time.Duration(c.Duration) * time.Millisecond)
		}
	}
}

// Start the main monitor container loop for a set amount of iterations
func Start(c *Conf) {
	log.Printf("starting docker-managedd\n------------------------------")
	a := &Action{Messages: []error{}}
	Monitor(c, a)
}
