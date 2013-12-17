// Print information about stale boards in your Trello org.
//
//   $ export KEY=...
//   $ export TOKEN=...
//   $ export ORG=...
//   $ go run tidytrello.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type BoardPrefs struct {
	PermissionLevel string `json:"permissionLevel"`
}

type Board struct {
	Id     string     `json:"id"`
	Url    string     `json:"url"`
	Closed bool       `json:"closed"`
	Prefs  BoardPrefs `json:"prefs"`
}

type BoardAction struct {
	Date string `json:"date"`
	Type string `json:"type"`
}

type BoardMember struct {
	Username string `json:"username"`
}

func mustGetenv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		panic("Missing " + k)
	}
	return v
}

var (
	trelloKey      = mustGetenv("KEY")
	trelloToken    = mustGetenv("TOKEN")
	trelloOrg      = mustGetenv("ORG")
	timeFormat     = "2006-01-02T15:04:05.999Z"
	threeMonths, _ = time.ParseDuration("2160h")
	threeMonthsAgo = time.Now().Add(-threeMonths)
)

func mustTimeParse(s string) time.Time {
	t, err := time.Parse(timeFormat, s)
	if err != nil {
		panic("Could not parse " + s)
	}
	return t
}

func mustGetTrello(r string, v interface{}) {
	fmt.Print(".")
	resp, err := http.Get("https://api.trello.com/1/" + r + "?key=" + trelloKey + "&token=" + trelloToken)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(v)
	if err != nil {
		panic(err)
	}
}

func main() {
	// Get list of boards
	boards := []Board{}
	mustGetTrello("organizations/" + trelloOrg + "/boards", &boards)

	// Filter to open and org-visible boards
	openBoards := []Board{}
	for _, board := range boards {
		if !board.Closed && board.Prefs.PermissionLevel == "org" {
			openBoards = append(openBoards, board)
		}
	}

	// Filter to stale boards
	staleBoards := []Board{}
	for _, board := range openBoards {
		actions := []BoardAction{}
		mustGetTrello("boards/" + board.Id + "/actions", &actions)
		var lastAction BoardAction
		for _, action := range actions {
			if action.Type != "makeAdminOfBoard" && action.Type != "addMemberToBoard" {
				lastAction = action
				break
			}
		}
		if (lastAction.Date == "") || threeMonthsAgo.After(mustTimeParse(lastAction.Date)) {
			staleBoards = append(staleBoards, board)
		}
	}

	// Print last newline after API call dots
	fmt.Println()

	// Print filtered boards
	for _, board := range staleBoards {
		members := []BoardMember{}
		mustGetTrello("boards/" + board.Id + "/members", &members)
		usernames := []string{}
		for _, member := range members {
			usernames = append(usernames, member.Username)
		}
		fmt.Printf("%s  (%s)\n", board.Url, strings.Join(usernames, ", "))
	}
}
