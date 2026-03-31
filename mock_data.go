package main

import "time"

func getMockEvents() []Event {
    return []Event{
        {
            ID: 1,
            HomeTeam: Team{Name: "Latvia"},
            AwayTeam: Team{Name: "Gibraltar"},
            HomeScore: Score{Current: 0},
            AwayScore: Score{Current: 0},
            Tournament: Tournament{Name: "UEFA Nations League", Category: Category{Name: "Europe"}},
            Status: Status{Description: "2nd half"},
            StatusTime: StatusTime{Timestamp: time.Now().Unix() - 600, Max: 5400, Initial: 2700, Extra: 540},
        },
        {
            ID: 2,
            HomeTeam: Team{Name: "Luxembourg"},
            AwayTeam: Team{Name: "Malta"},
            HomeScore: Score{Current: 3},
            AwayScore: Score{Current: 0},
            Tournament: Tournament{Name: "UEFA Nations League", Category: Category{Name: "Europe"}},
            Status: Status{Description: "2nd half"},
            StatusTime: StatusTime{Timestamp: time.Now().Unix() - 120, Max: 5400, Initial: 2700, Extra: 540},
        },
    }
}

func getMockIncidents(id int64) []Incident {
    if id == 2 {
        return []Incident{
            {IncidentType: "card", CardType: "Red", Player: Player{Name: "Borg"}, IncidentClass: "away"},
            {IncidentType: "card", CardType: "Yellow", Player: Player{Name: "Chanot"}, IncidentClass: "home"},
        }
    }
    return nil
}
