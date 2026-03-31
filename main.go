package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item struct {
	event Event
}

func (i item) Title() string       { return fmt.Sprintf("%s %d - %d %s", i.event.HomeTeam.Name, i.event.HomeScore.Current, i.event.AwayScore.Current, i.event.AwayTeam.Name) }
func (i item) Description() string { return i.event.Status.Description }
func (i item) FilterValue() string { return i.event.HomeTeam.Name + " " + i.event.AwayTeam.Name }

type model struct {
	list     list.Model
	selected *Event
	err      error
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "enter" {
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.selected = &i.event
				return m, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

func main() {
	s := NewScraper()
	var events []Event
	var err error
	if os.Getenv("USE_MOCK") == "1" {
		events = getMockEvents()
	} else {
		events, err = s.GetLiveEvents()
	}
	if err != nil && os.Getenv("USE_MOCK") != "1" {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Use USE_MOCK=1 to run with mock data.")
		os.Exit(1)
	}

	groups := GroupByLeague(events)
	leagues := GetLeagueNames(groups)

	var items []list.Item
	for _, l := range leagues {
		for _, e := range groups[l] {
			items = append(items, item{event: e})
		}
	}

	m := model{
		list: list.New(items, list.NewDefaultDelegate(), 0, 0),
	}
	m.list.Title = "Live Soccer Games"

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}

	res := finalModel.(model)
	if res.selected != nil {
		pollGame(res.selected.ID)
	}
}

func pollGame(eventID int64) {
	s := NewScraper()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		var event *Event
		var err error
		if os.Getenv("USE_MOCK") == "1" {
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
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else if event == nil {
			fmt.Fprintf(os.Stderr, "Event not found\n")
		} else {
			var incidents []Incident
			var err error
			if os.Getenv("USE_MOCK") == "1" {
				incidents = getMockIncidents(eventID)
			} else {
				incidents, err = s.GetIncidents(eventID)
			}

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			} else {
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
					out.Minute = fmt.Sprintf("%d", currentMin+1) // typically show next minute in football
				}

				var reds []string
				var yellows []string
				for _, inc := range incidents {
					if inc.IncidentType == "card" {
						team := event.HomeTeam.Name
						if inc.IncidentClass == "away" {
							team = event.AwayTeam.Name
						}
						cardStr := fmt.Sprintf("%s (%s)", inc.Player.Name, team)
						if inc.CardType == "Red" || inc.CardType == "YellowRed" {
							reds = append(reds, cardStr)
						} else if inc.CardType == "Yellow" {
							yellows = append(yellows, cardStr)
						}
					}
				}
				if len(reds) > 0 {
					out.RedCards = strings.Join(reds, ", ")
				}
				if len(yellows) > 0 {
					out.YellowCards = strings.Join(yellows, ", ")
				}

				b, _ := json.MarshalIndent(out, "", "  ")
				fmt.Print("\033[H\033[2J")
				fmt.Println(string(b))
			}
		}

        if os.Getenv("USE_MOCK") == "1" {
            break
        }
		<-ticker.C
	}
}

type JsonOutput struct {
	Score       string `json:"score"`
	Minute      string `json:"minute"`
	RedCards    string `json:"redCards,omitempty"`
	YellowCards string `json:"yellowCards,omitempty"`
}
