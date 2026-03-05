package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/gocolly/colly/v2"
)

// Lift represents a single bridge lift event
type Lift struct {
	FullDate   string `json:"full_date"`
	Day        string `json:"day"`
	Date       int    `json:"date"`
	Month      string `json:"month"`
	Year       int    `json:"year"`
	DateDMY    string `json:"date_dmy"`
	Time       string `json:"time"`
	Vessel     string `json:"vessel"`
	Direction  string `json:"direction"`
	VesselType string `json:"vessel_type"`
}

var months = map[string]int{
	"January": 1, "February": 2, "March": 3, "April": 4,
	"May": 5, "June": 6, "July": 7, "August": 8,
	"September": 9, "October": 10, "November": 11, "December": 12,
}

func parseDate(dateStr string) (day string, d int, month string, y int, dmy string) {
	// Format: "Saturday 17 October 2026"
	parts := strings.Fields(dateStr)
	if len(parts) >= 4 {
		day = parts[0]
		fmt.Sscanf(parts[1], "%d", &d)
		month = parts[2]
		fmt.Sscanf(parts[3], "%d", &y)

		// Case-insensitive month lookup
		monthNum := months[strings.Title(strings.ToLower(month))]
		if monthNum == 0 {
			// Fallback if strings.Title isn't enough (e.g. non-standard casing)
			for mName, mNum := range months {
				if strings.EqualFold(mName, month) {
					monthNum = mNum
					break
				}
			}
		}

		dmy = fmt.Sprintf("%02d/%02d/%d", d, monthNum, y)
	}
	return
}

func main() {
	// Instantiate default collector
	c := colly.NewCollector(
		colly.AllowedDomains("www.towerbridge.org.uk"),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
	)

	var lifts []Lift

	// On every .time-table element (which groups lifts by date)
	c.OnHTML(".time-table", func(e *colly.HTMLElement) {
		dateStr := strings.TrimSpace(e.ChildText(".time-table__heading"))

		// Iterate over each lift row within the table
		e.ForEach(".bridge-lift-row", func(_ int, row *colly.HTMLElement) {
			day, d, month, y, dmy := parseDate(dateStr)
			lift := Lift{
				FullDate: dateStr,
				Day:      day,
				Date:     d,
				Month:    month,
				Year:     y,
				DateDMY:  dmy,
			}

			// The details are held within paragraphs in the content div
			row.ForEach(".bridge-lift-row__content p", func(i int, p *colly.HTMLElement) {
				text := strings.TrimSpace(p.Text)

				// Pattern observed:
				// i=0: Time and Direction (Time is usually in <strong>)
				// i=1: Vessel Type
				// i=2: Vessel Name (Name is usually in <strong>)
				switch i {
				case 0:
					lift.Time = strings.TrimSpace(p.ChildText("strong"))
					// Extract direction by removing time from full text
					direction := strings.TrimSpace(strings.Replace(text, lift.Time, "", 1))
					lift.Direction = direction
				case 1:
					lift.VesselType = text
				case 2:
					vessel := strings.TrimSpace(p.ChildText("strong"))
					if vessel == "" {
						vessel = text
					}
					lift.Vessel = vessel
				}
			})

			if lift.Time != "" || lift.Vessel != "" {
				lifts = append(lifts, lift)
			}
		})
	})

	// Before making a request print "Visiting ..."
	c.OnRequest(func(r *colly.Request) {
		log.Printf("Visiting %s\n", r.URL)
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %s failed with response: %v\nError: %v\n", r.Request.URL, r, err)
	})

	// Start scraping
	err := c.Visit("https://www.towerbridge.org.uk/bridge-lifts")
	if err != nil {
		log.Fatalf("Could not visit page: %v", err)
	}

	// Convert results to JSON
	jsonData, err := json.MarshalIndent(lifts, "", "  ")
	if err != nil {
		log.Fatalf("JSON marshaling failed: %v", err)
	}

	// Output the JSON
	fmt.Println(string(jsonData))
}
