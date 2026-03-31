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
    // In WASM mode, we'll bypass the TUI and just let the user monitor a specific game
    // or provide a simple list via fmt.Printf if we can.

    // For now, let's just make a simple polling loop that doesn't use bubbletea.
    // If the user wants to select a game, they can pass an ID.

    eventIDStr := os.Getenv("EVENT_ID")
    if eventIDStr == "" {
        // List live games and exit
        listGames()
        return
    }

    pollGame(eventIDStr)
}

func listGames() {
    s := NewScraper()
    events, err := s.GetLiveEvents()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    groups := GroupByLeague(events)
    leagues := GetLeagueNames(groups)

    fmt.Println("LIVE GAMES (Set EVENT_ID to monitor):")
    for _, l := range leagues {
        fmt.Printf("\n--- %s ---\n", l)
        for _, e := range groups[l] {
            fmt.Printf("[%d] %s %d - %d %s\n", e.ID, e.HomeTeam.Name, e.HomeScore.Current, e.AwayScore.Current, e.AwayTeam.Name)
        }
    }
}

func pollGame(eventIDStr string) {
    var eventID int64
    fmt.Sscanf(eventIDStr, "%d", &eventID)

    s := NewScraper()
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        event, err := s.GetEventDetails(eventID)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
        } else {
            incidents, _ := s.GetIncidents(eventID)
            out := JsonOutput{
                Score: fmt.Sprintf("%s %d-%d %s", event.HomeTeam.Name, event.HomeScore.Current, event.AwayScore.Current, event.AwayTeam.Name),
            }

            // Minute logic
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
            out.RedCards = strings.Join(reds, ", ")
            out.YellowCards = strings.Join(yellows, ", ")

            b, _ := json.MarshalIndent(out, "", "  ")
            fmt.Println(string(b))
        }
        <-ticker.C
    }
}
