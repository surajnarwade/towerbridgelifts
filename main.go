package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"os"
	"path/filepath"

	"github.com/gocolly/colly/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

// LiftData wraps the lifts with metadata
type LiftData struct {
	LastUpdated string `json:"last_updated"`
	Lifts       []Lift `json:"lifts"`
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

		// Case-insensitive month lookup using cases.Title (replacing deprecated strings.Title)
		caser := cases.Title(language.English)
		monthNum := months[caser.String(strings.ToLower(month))]
		if monthNum == 0 {
			// Fallback if title casing isn't enough
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

func pruneOldFiles(dir string, days int) {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("Warning: could not read data directory for pruning: %v", err)
		return
	}

	now := time.Now()
	cutoff := now.AddDate(0, 0, -days)

	prunedCount := 0
	for _, f := range files {
		if f.IsDir() || !strings.HasPrefix(f.Name(), "lifts_") || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}

		info, err := f.Info()
		if err != nil {
			log.Printf("Warning: could not get file info for %s: %v", f.Name(), err)
			continue
		}

		if info.ModTime().Before(cutoff) {
			err := os.Remove(filepath.Join(dir, f.Name()))
			if err == nil {
				prunedCount++
			} else {
				log.Printf("Warning: could not remove old file %s: %v", f.Name(), err)
			}
		}
	}
	if prunedCount > 0 {
		log.Printf("Pruned %d files older than %d days\n", prunedCount, days)
	}
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

	// Safety check: if no lifts found, maybe the layout changed or site is down
	if len(lifts) == 0 {
		log.Println("No lifts found. Site layout might have changed or no lifts scheduled.")
		// Don't exit with error, just don't overwrite if you want to keep old data,
		// but usually we want to know it failed.
		os.Exit(0)
	}

	// Wrap in data structure with timestamp
	data := LiftData{
		LastUpdated: time.Now().Format("2006-01-02 15:04:05"),
		Lifts:       lifts,
	}

	// Convert results to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("JSON marshaling failed: %v", err)
	}

	// Ensure data directory exists
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Prune files older than 30 days
	pruneOldFiles(dataDir, 30)

	// Create a unique timestamped filename
	timestamp := time.Now().Format("20060102_150405")
	timestampedFilename := fmt.Sprintf("lifts_%s.json", timestamp)
	timestampedPath := filepath.Join(dataDir, timestampedFilename)

	// Write the JSON to the timestamped file
	if err := os.WriteFile(timestampedPath, jsonData, 0644); err != nil {
		log.Fatalf("Failed to write timestamped JSON file: %v", err)
	}

	// Also update 'latest.json' for the frontend
	latestPath := filepath.Join(dataDir, "latest.json")
	if err := os.WriteFile(latestPath, jsonData, 0644); err != nil {
		log.Fatalf("Failed to write latest.json: %v", err)
	}

	log.Printf("Successfully scraped %d lifts.\n", len(lifts))
	log.Printf("Saved to: %s\n", timestampedPath)
	log.Printf("Updated: %s\n", latestPath)
}
