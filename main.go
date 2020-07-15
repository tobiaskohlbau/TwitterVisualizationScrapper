package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

type TimelineItem struct {
	ID        int64  `json:"id"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
}

type DayDate struct {
	Day   int
	Month int
	Year  int
}

func (d DayDate) String() string {
	return fmt.Sprintf("%d.%d.%d", d.Day, d.Month, d.Year)
}

func main() {
	if _, err := os.Stat("input.csv"); errors.Is(err, os.ErrNotExist) {
		log.Fatal("please provide a input.csv with contains a twitter handle per line")
	}

	f, err := os.Open("input.csv")
	if err != nil {
		log.Fatalf("failed to open input file: %s", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		username := scanner.Text()
		userTimeline, err := fetchUser(username)
		if err != nil {
			log.Fatalf("failed to fetch twitter timeline for user %s: %s", username, err)
		}

		err = plotUser(username, userTimeline)
		if err != nil {
			log.Fatalf("failed to plot twitter timeline for user %s: %s", username, err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("failed to scan input: %s", err)
	}
}

func fetchUser(user string) ([]TimelineItem, error) {
	config := &clientcredentials.Config{
		ClientID:     os.Getenv("TWITTER_CLIENTID"),
		ClientSecret: os.Getenv("TWITTER_CLIENTSECRET"),
		TokenURL:     "https://api.twitter.com/oauth2/token",
	}
	twitterClient := config.Client(oauth2.NoContext)

	requestURL := fmt.Sprintf("https://api.twitter.com/1.1/statuses/user_timeline.json?screen_name=%s&exclude_replies=true&include_rts=false&count=200", user)
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	res, err := twitterClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch timelime: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got bad response: %s", res.Status)
	}

	timelineResponse := []TimelineItem{}
	if err := json.NewDecoder(res.Body).Decode(&timelineResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return timelineResponse, nil
}

func plotUser(username string, data []TimelineItem) error {
	p, err := plot.New()
	if err != nil {
		return fmt.Errorf("failed to plot user: %w", err)
	}

	tweetsPerDay := make(map[DayDate]int)
	for _, item := range data {
		t, err := time.Parse(time.RubyDate, item.CreatedAt)
		if err != nil {
			return fmt.Errorf("invalid date: %w", err)
		}
		date := DayDate{Day: t.Day(), Month: int(t.Month()), Year: t.Year()}

		tweetsPerDay[date] = tweetsPerDay[date] + 1
	}

	keys := make([]DayDate, 0, len(tweetsPerDay))
	for k := range tweetsPerDay {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Year < keys[j].Year {
			return true
		}
		if keys[i].Month < keys[j].Month && keys[i].Year <= keys[j].Year {
			return true
		}
		if keys[i].Day < keys[j].Day && keys[i].Month <= keys[j].Month && keys[i].Year <= keys[j].Year {
			return true
		}
		return false
	})

	values := plotter.Values{}
	verticalLabels := []string{}
	for _, k := range keys {
		values = append(values, float64(tweetsPerDay[k]))
		verticalLabels = append(verticalLabels, k.String())
	}

	barChart, err := plotter.NewBarChart(values, 0.1*vg.Centimeter)
	if err != nil {
		return fmt.Errorf("failed to create barchart: %w", err)
	}

	p.Add(barChart)

	ticks := make([]plot.Tick, len(verticalLabels))
	for i, name := range verticalLabels {
		ticks[i] = plot.Tick{Value: float64(i), Label: name}
	}
	p.X.Tick.Marker = plot.ConstantTicks(ticks)
	p.X.Tick.Label.Rotation = 1.1
	p.X.Tick.Label.XAlign = -1.04
	p.X.Padding = 30

	if _, err := os.Stat("output"); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir("output", 0744); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	err = p.Save(1024, 512, filepath.Join("output", username+".png"))
	if err != nil {
		return fmt.Errorf("failed to save plot: %w", err)
	}

	return nil
}
