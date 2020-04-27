package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

var (
	WebhookUrl = os.Getenv("WEBHOOK_URL")
)

func handleEvent(e events.Message) {
	log.Printf("action:%s %v", e.Action, e.Actor.Attributes)

	name, ok := e.Actor.Attributes["name"]
	if !ok {
		return
	}
	action := e.Action
	switch {
	case action == "start":
		action = "is starting"
	case action == "die":
		action = "is exiting"
	case strings.Contains(action, "exec_start"):
		action = "is exec-ing " + strings.TrimSpace(action[12:len(action)])
	default:
		log.Printf("unsupported action: %+v", e)
		return
	}

	tmpl := "*%s* %s:\n```\n%v\n```"
	buf, _ := json.Marshal(map[string]string{
		"text": fmt.Sprintf(tmpl, name, action, e.Actor.Attributes),
	})

	resp, err := http.Post(WebhookUrl, "application/json", bytes.NewBuffer(buf))
	if err != nil {
		log.Printf("error submitting webhook: %s", err)
		return
	}
	log.Printf("sent webhook with status %d", resp.StatusCode)
}

func main() {
	if WebhookUrl == "" {
		log.Fatal("missing WEBHOOK_URL env variable")
		return
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("error creating client: %s", err)
	}
	info, err := cli.Info(context.Background())
	if err != nil {
		log.Fatalf("error fetching server info: %s", err)
	}
	log.Printf("connected to docker api: %s", info.Name)

	filter := filters.NewArgs()
	filter.Add("type", "container")
	filter.Add("event", "start")
	filter.Add("event", "die")
	filter.Add("event", "exec_start")

	msgChan, errChan := cli.Events(context.Background(), types.EventsOptions{
		Filters: filter,
	})
	log.Printf("listening for events")

	for {
		select {
		case err := <-errChan:
			log.Fatalf("error reading events: %s", err)
		case e := <-msgChan:
			handleEvent(e)
		}
	}
}
