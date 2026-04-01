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

    proxy := os.Getenv("CORS_PROXY")
    if proxy != "" {
        fmt.Printf("Using CORS Proxy: %s\n", proxy)
    } else {
        fmt.Println("Warning: No CORS_PROXY set. Requests will likely fail in browser.")
    }

    eventIDStr := os.Getenv("EVENT_ID")
    useMock := os.Getenv("USE_MOCK") == "1"

    if eventIDStr == "" {
        listGames(useMock)
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
        fmt.Println("Using mock data...")
        events = getMockEvents()
    } else {
        fmt.Println("Fetching games from SofaScore...")
        events, err = s.GetLiveEvents()
    }

    if err != nil {
        fmt.Printf("\nError fetching events: %v\n", err)
        fmt.Println("Check if the proxy is working and if you have authorized access at https://cors-anywhere.herokuapp.com/corsdemo")
        return
    }

    if len(events) == 0 {
        fmt.Println("\nNo games found (live or scheduled for today).")
        return
    }

    fmt.Printf("\nFound %d games.\n", len(events))
    groups := GroupByLeague(events)
    leagues := GetLeagueNames(groups)

    for _, l := range leagues {
        fmt.Printf("\n[%s]\n", l)
        for _, e := range groups[l] {
            statusPrefix := ""
            if e.Status.Type == "inprogress" {
                statusPrefix = "* LIVE * "
            }
            fmt.Printf("  ID: %d | %s%s %d - %d %s (%s)\n", e.ID, statusPrefix, e.HomeTeam.Name, e.HomeScore.Current, e.AwayScore.Current, e.AwayTeam.Name, e.Status.Description)
        }
    }
    fmt.Println("\nTo monitor a specific game, refresh the page with ?id=ID")
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
            fmt.Print("\033[H\033[2J")
            fmt.Println(string(b))
        }
        <-ticker.C
    }
}
