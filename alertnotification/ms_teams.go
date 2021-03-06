// https://github.com/rakutentech/go-alertnotification/blob/master/ms_teams.go

package alertnotification

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"time"
)

// MsTeam is MessageCard for Team notification
type MsTeam struct {
	Type       string          `json:"@type"`
	Context    string          `json:"@context"`
	Summary    string          `json:"summary"`
	ThemeColor string          `json:"themeColor"`
	Title      string          `json:"title"`
	Sections   []SectionStruct `json:"sections"`
}

// SectionStruct is sub-struct of MsTeam
type SectionStruct struct {
	ActivityTitle    string       `json:"activityTitle"`
	ActivitySubtitle string       `json:"activitySubtitle"`
	ActivityImage    string       `json:"activityImage"`
	Facts            []FactStruct `json:"facts"`
}

// FactStruct is sub-struct of SectionStruct
type FactStruct struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// NewMsTeam is used to create MsTeam
func NewMsTeam(color, title, activitySubTitle string, attributes map[string]string) *MsTeam {
	facts := make([]FactStruct, 0)
	for k, v := range attributes {
		facts = append(facts, FactStruct{Name: k, Value: v})
	}

	sort.Slice(facts, func(i, j int) bool {
		return facts[i].Name < facts[j].Name
	})

	hostname, _ := os.Hostname()
	facts = append([]FactStruct{FactStruct{Name: "Hostname", Value: hostname}}, facts...)

	notificationCard := MsTeam{
		Type:       "MessageCard",
		Context:    "http://schema.org/extensions",
		Summary:    os.Getenv("MS_TEAMS_CARD_SUBJECT"),
		ThemeColor: color,
		Title:      title + " " + activitySubTitle,
		Sections: []SectionStruct{
			SectionStruct{
				ActivityTitle:    title + " " + activitySubTitle,
				ActivitySubtitle: activitySubTitle,
				ActivityImage:    "",
				Facts:            facts,
			},
		},
	}

	return &notificationCard
}

// Send is implementation of interface AlertNotification's Send()
func (card *MsTeam) Send() error {
	requestBody, err := json.Marshal(card)
	if err != nil {
		return err
	}

	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	wb := os.Getenv("MS_TEAMS_WEBHOOK")
	if len(wb) == 0 {
		return errors.New("Cannot sent alert to MsTeams. MS_TEAMS_WEBHOOK is not set in the environment. ")
	}
	request, err := http.NewRequest("POST", wb, bytes.NewBuffer(requestBody))
	request.Header.Set("Content-type", "application/json")
	if err != nil {
		return err
	}

	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(respBody) != "1" {
		return fmt.Errorf("Teams WebHook returned %s", respBody)
	}
	return nil
}

// DeferSend send the card in the background
func (card *MsTeam) DeferSend() error {
	go func() {
		if err := card.Send(); err != nil {
			log.Panicln(err)
		}
	}()

	return nil
}
