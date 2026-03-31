package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type JsonOutput struct {
	Score       string `json:"score"`
	Minute      string `json:"minute"`
	RedCards    string `json:"redCards,omitempty"`
	YellowCards string `json:"yellowCards,omitempty"`
}

func main() {
    fmt.Println("gol.json starting...")

    eventIDStr := os.Getenv("EVENT_ID")
    useMock := os.Getenv("USE_MOCK") == "1"

    if eventIDStr == "" {
        fmt.Println("No EVENT_ID provided. Listing live games...")
        listGames(useMock)
        // In WASM, if main exits, the program stops.
        // We should probably keep it alive if we want to support dynamic interactions later,
        // but for a list it's fine to exit or just wait.
        select {}
    } else {
        fmt.Printf("Monitoring Event ID: %s\n", eventIDStr)
        pollGame(eventIDStr, useMock)
    }
}

func listGames(useMock bool) {
    s := NewScraper()
    var events []Event
    var err error

    if useMock {
        events = getMockEvents()
    } else {
        fmt.Println("Fetching live events from SofaScore (via proxy)...")
        events, err = s.GetLiveEvents()
    }

    if err != nil {
        fmt.Printf("Error fetching events: %v\n", err)
        return
    }

    if len(events) == 0 {
        fmt.Println("No live games found.")
        return
    }

    groups := GroupByLeague(events)
    leagues := GetLeagueNames(groups)

    for _, l := range leagues {
        fmt.Printf("\n[%s]\n", l)
        for _, e := range groups[l] {
            fmt.Printf("  ID: %d | %s %d - %d %s\n", e.ID, e.HomeTeam.Name, e.HomeScore.Current, e.AwayScore.Current, e.AwayTeam.Name)
        }
    }
    fmt.Println("\nTo monitor a specific game, refresh with ?id=ID")
}

func pollGame(eventIDStr string, useMock bool) {
    var eventID int64
    fmt.Sscanf(eventIDStr, "%d", &eventID)

    s := NewScraper()
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        var event *Event
        var err error

        if useMock {
            es := getMockEvents()
            for _, e := range es {
                if e.ID == eventID {
                    event = &e
                    break
                }
            }
        } else {
            event, err = s.GetEventDetails(eventID)
        }

        if err != nil {
            fmt.Printf("Error fetching details: %v\n", err)
        } else if event == nil {
            fmt.Printf("Event ID %d not found.\n", eventID)
        } else {
            var incidents []Incident
            if useMock {
                incidents = getMockIncidents(eventID)
            } else {
                incidents, _ = s.GetIncidents(eventID)
            }

            out := JsonOutput{
                Score: fmt.Sprintf("%s %d-%d %s", event.HomeTeam.Name, event.HomeScore.Current, event.AwayScore.Current, event.AwayTeam.Name),
            }

            now := time.Now().Unix()
            elapsed := int(now - event.StatusTime.Timestamp)
            if elapsed < 0 { elapsed = 0 }
            currentMin := (event.StatusTime.Initial + elapsed) / 60
            maxRegular := event.StatusTime.Max / 60

            if currentMin >= maxRegular {
                added := currentMin - maxRegular
                totalAdded := event.StatusTime.Extra / 60
                if totalAdded > 0 {
                    out.Minute = fmt.Sprintf("%d+%d (+%d)", maxRegular, added, totalAdded)
                } else {
                    out.Minute = fmt.Sprintf("%d+%d", maxRegular, added)
                }
            } else {
                out.Minute = fmt.Sprintf("%d", currentMin+1)
            }

            var reds, yellows []string
            for _, inc := range incidents {
                if inc.IncidentType == "card" {
                    team := event.HomeTeam.Name
                    if inc.IncidentClass == "away" { team = event.AwayTeam.Name }
                    cardStr := fmt.Sprintf("%s (%s)", inc.Player.Name, team)
                    if inc.CardType == "Red" || inc.CardType == "YellowRed" {
                        reds = append(reds, cardStr)
                    } else if inc.CardType == "Yellow" {
                        yellows = append(yellows, cardStr)
                    }
                }
            }
            if len(reds) > 0 { out.RedCards = strings.Join(reds, ", ") }
            if len(yellows) > 0 { out.YellowCards = strings.Join(yellows, ", ") }

            b, _ := json.MarshalIndent(out, "", "  ")
            // Clear screen (ANSI)
            fmt.Print("\033[H\033[2J")
            fmt.Println(string(b))
        }
        <-ticker.C
    }
}
