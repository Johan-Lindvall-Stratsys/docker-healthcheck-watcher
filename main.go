package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"github.com/bytearena/docker-healthcheck-watcher/alertnotification"
	"github.com/stratsys/go-common/env"
)

var (
	showHealthForContainerIDs map[string]bool               = make(map[string]bool)
	deadServiceIDs            map[string]time.Time          = make(map[string]time.Time)
	updatingServiceIDS        map[string]time.Time          = make(map[string]time.Time)
	logWatchers               map[string]context.CancelFunc = make(map[string]context.CancelFunc)
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

func onLogStdErr(msg, service string, attributes map[string]string) {
	if err := alertnotification.NewMsTeam("ff5864", service, "logged "+msg, attributes).DeferSend(); err != nil {
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

	stream, errChan := cli.Events(ctx, types.EventsOptions{})

	filter := filters.NewArgs()
	filter.Add("label", fmt.Sprintf("com.docker.swarm.service.name=%s", os.Getenv("STDERR_SERVICE")))

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
		Filters: filter,
	})

	if err != nil {
		log.Panicln(clientError)
	}

	for _, container := range containers {
		container.Labels["container_id"] = container.ID
		startLogWatch(cli, container.ID, container.Labels)
	}

	for {
		select {
		case msg := <-errChan:
			log.Panicln(msg)
		case msg := <-stream:
			handleMessage(cli, msg)
		}
	}
}

func startLogWatch(cli *client.Client, id string, attributes map[string]string) {
	if _, ok := logWatchers[id]; !ok {
		fmt.Printf("Starting log watch for %s\n", id)
		ctx, cancel := context.WithCancel(context.Background())
		logWatchers[id] = cancel

		go func() {
			reader, err := cli.ContainerLogs(ctx, id, types.ContainerLogsOptions{ShowStderr: true, Follow: true, Since: "0s"})
			if err != nil {
				return
			}
			header := make([]byte, 8)
			for {
				if _, err := io.ReadFull(reader, header); err != nil {
					break
				}
				count := binary.BigEndian.Uint32(header[4:])
				data := make([]byte, count)
				if _, err = io.ReadFull(reader, data); err == nil {
					onLogStdErr(strings.TrimSpace(string(data)), getServiceName(attributes), attributes)
				} else {
					break
				}
			}
		}()
	}
}

func stopLogWatch(id string) {
	if cancel, ok := logWatchers[id]; ok {
		fmt.Printf("Stopping log watch for %s\n", id)
		delete(logWatchers, id)
		cancel()
	}
}

func handleMessage(cli *client.Client, msg events.Message) {
	if msg.Type == "service" && msg.Action == "update" {
		if new, ok := msg.Actor.Attributes["updatestate.new"]; ok {
			if new == "updating" {
				updatingServiceIDS[msg.Actor.ID] = time.Now()
			} else {
				delete(updatingServiceIDS, msg.Actor.ID)
			}
		}
	} else if msg.Type == events.ContainerEventType {
		msg.Actor.Attributes["container_id"] = msg.Actor.ID
		if msg.Action == "start" {
			showHealthForContainerIDs[msg.Actor.ID] = false
			if getServiceName(msg.Actor.Attributes) == os.Getenv("STDERR_SERVICE") {
				startLogWatch(cli, msg.Actor.ID, msg.Actor.Attributes)
			}
		} else if msg.Action == "health_status: unhealthy" {
			showHealthForContainerIDs[msg.Actor.ID] = true
			onContainerHealthCheckFailure(getServiceName(msg.Actor.Attributes), msg.Actor.Attributes)
		} else if msg.Action == "health_status: healthy" {
			if showHealth, ok := showHealthForContainerIDs[msg.Actor.ID]; !ok || showHealth {
				onContainerHealthy(getServiceName(msg.Actor.Attributes), msg.Actor.Attributes)
			}
		} else if msg.Action == "die" {
			stopLogWatch(msg.Actor.ID)
			serviceID := getServiceID(msg.Actor.Attributes)
			exitCode := msg.Actor.Attributes["exitCode"]

			if exitCode != "0" {
				now := time.Now()
				if updating, ok := updatingServiceIDS[serviceID]; ok && exitCode == "1" && now.Sub(updating) < 2*time.Minute {
					delete(updatingServiceIDS, serviceID)
				} else if death, ok := deadServiceIDs[serviceID]; !ok || now.Sub(death) > 2*time.Minute {
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
