package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"
)

type EventResponse struct {
	Events []Event `json:"events"`
}

type Event struct {
	ID           int64        `json:"id"`
	HomeTeam     Team         `json:"homeTeam"`
	AwayTeam     Team         `json:"awayTeam"`
	HomeScore    Score        `json:"homeScore"`
	AwayScore    Score        `json:"awayScore"`
	Tournament   Tournament   `json:"tournament"`
	Status       Status       `json:"status"`
	LastPeriod   string       `json:"lastPeriod"`
	StatusTime   StatusTime   `json:"statusTime"`
}

type Team struct {
	Name string `json:"name"`
}

type Score struct {
	Current int `json:"current"`
}

type Tournament struct {
	Name string `json:"name"`
	Category Category `json:"category"`
}

type Category struct {
	Name string `json:"name"`
}

type Status struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type StatusTime struct {
	Timestamp int64 `json:"timestamp"`
	Max       int   `json:"max"`
	Initial   int   `json:"initial"`
	Extra     int   `json:"extra"`
}

type IncidentResponse struct {
	Incidents []Incident `json:"incidents"`
}

type Incident struct {
	Type          string `json:"type"`
	Player        Player `json:"player"`
	IncidentClass string `json:"incidentClass"` // "home" or "away"
	IncidentType  string `json:"incidentType"`  // "card", "goal", etc.
	CardType      string `json:"cardType"`      // "Yellow", "Red", "YellowRed"
	Time          int    `json:"time"`
	AddedTime     int    `json:"addedTime"`
}

type Player struct {
	Name string `json:"name"`
}

type Scraper struct {
	client *http.Client
	proxy  string
}

func NewScraper() *Scraper {
    proxy := os.Getenv("CORS_PROXY")
    if proxy == "" {
        // Fallback or user can set it
        // proxy = "https://cors-anywhere.herokuapp.com/"
    }
	return &Scraper{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
        proxy: proxy,
	}
}

func (s *Scraper) doRequest(url string) (*http.Response, error) {
    targetURL := url
    if s.proxy != "" {
        targetURL = s.proxy + url
    }
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.sofascore.com/")
	req.Header.Set("Origin", "https://www.sofascore.com")

	return s.client.Do(req)
}

func (s *Scraper) GetLiveEvents() ([]Event, error) {
	resp, err := s.doRequest("https://api.sofascore.com/api/v1/sport/football/events/live")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("access denied (403 Forbidden)")
	}

	var data EventResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data.Events, nil
}

func (s *Scraper) GetIncidents(eventID int64) ([]Incident, error) {
	url := fmt.Sprintf("https://api.sofascore.com/api/v1/event/%d/incidents", eventID)
	resp, err := s.doRequest(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data IncidentResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data.Incidents, nil
}

func (s *Scraper) GetEventDetails(eventID int64) (*Event, error) {
	url := fmt.Sprintf("https://api.sofascore.com/api/v1/event/%d", eventID)
	resp, err := s.doRequest(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Event Event `json:"event"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &data.Event, nil
}

func GroupByLeague(events []Event) map[string][]Event {
	groups := make(map[string][]Event)
	for _, e := range events {
		league := fmt.Sprintf("%s: %s", e.Tournament.Category.Name, e.Tournament.Name)
		groups[league] = append(groups[league], e)
	}
	return groups
}

func GetLeagueNames(groups map[string][]Event) []string {
	leagues := make([]string, 0, len(groups))
	for l := range groups {
		leagues = append(leagues, l)
	}
	sort.Strings(leagues)
	return leagues
}
