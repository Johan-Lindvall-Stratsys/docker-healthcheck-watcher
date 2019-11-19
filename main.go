package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"

	"github.com/bytearena/docker-healthcheck-watcher/alertnotification"
	"github.com/stratsys/go-common/env"
)

var (
	showHealthForContainerIDs map[string]bool      = make(map[string]bool)
	deadServiceIDs            map[string]time.Time = make(map[string]time.Time)
)

func onContainerDieFailure(service string, exitCode string, attributes map[string]string) {
	if err := alertnotification.NewMsTeam("ff5864", service, "died (exited with code "+exitCode+")", attributes).DeferSend(); err != nil {
		log.Panicln(err)
	}
}

func onContainerHealthy(service string, attributes map[string]string) {
	if err := alertnotification.NewMsTeam("90ee90", service, "ok", attributes).DeferSend(); err != nil {
		log.Panicln(err)
	}
}

func onContainerHealthCheckFailure(service string, attributes map[string]string) {
	if err := alertnotification.NewMsTeam("ff5864", service, "unhealthy (running)", attributes).DeferSend(); err != nil {
		log.Panicln(err)
	}
}

func main() {
	kvps := make(map[string]string)
	for _, file := range os.Args[1:] {
		if err := env.ReadFiles(file, kvps); err != nil {
			panic(err)
		}
	}

	for k, v := range kvps {
		os.Setenv(k, v)
	}

	cli, clientError := client.NewEnvClient()
	ctx := context.Background()

	if clientError != nil {
		log.Panicln(clientError)
	}

	stream, err := cli.Events(ctx, types.EventsOptions{})

	for {
		select {
		case msg := <-err:
			log.Panicln(msg)
		case msg := <-stream:
			handleMessage(msg)
		}
	}
}

func handleMessage(msg events.Message) {
	if msg.Type == events.ContainerEventType {
		msg.Actor.Attributes["container_id"] = msg.Actor.ID
		if msg.Action == "start" {
			showHealthForContainerIDs[msg.Actor.ID] = false
		} else if msg.Action == "health_status: unhealthy" {
			showHealthForContainerIDs[msg.Actor.ID] = true
			onContainerHealthCheckFailure(getServiceName(msg.Actor.Attributes), msg.Actor.Attributes)
		} else if msg.Action == "health_status: healthy" {
			if showHealth, ok := showHealthForContainerIDs[msg.Actor.ID]; !ok || showHealth {
				onContainerHealthy(getServiceName(msg.Actor.Attributes), msg.Actor.Attributes)
			}
		} else if msg.Action == "die" {
			serviceID := getServiceID(msg.Actor.Attributes)
			exitCode := msg.Actor.Attributes["exitCode"]

			if exitCode != "0" {
				now := time.Now()
				if death, ok := deadServiceIDs[serviceID]; !ok || now.Sub(death) > 2*time.Minute {
					deadServiceIDs[serviceID] = now
					onContainerDieFailure(getServiceName(msg.Actor.Attributes), exitCode, msg.Actor.Attributes)
				}
			}
		}
	}
}

func getServiceName(attributes map[string]string) string {
	if nm, ok := attributes["com.docker.swarm.service.name"]; ok {
		return nm
	}

	return attributes["image"]
}

func getServiceID(attributes map[string]string) string {
	if id, ok := attributes["com.docker.swarm.service.id"]; ok {
		return id
	}

	return ""
}
