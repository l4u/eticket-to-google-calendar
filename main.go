package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
)

type Flight struct {
	FlightNumber  string
	StartDateTime string
	StartLoc      string
	EndDateTime   string
	EndLoc        string
}

func formattedDateTime(inputDateTime string, inputLoc string) (time.Time, error) {
	loc, err := time.LoadLocation(inputLoc)
	if err != nil {
		return time.Time{}, err
	}
	ansic := "Jan _2 15:04 2006"
	result, err := time.ParseInLocation(ansic, inputDateTime, loc)
	if err != nil {
		return time.Time{}, err
	}

	return result, nil
}

func locationToZone(loc string) string {

	switch loc {
	case "Hong Kong":
		return "Asia/Hong_Kong"
	case "Tokyo(Haneda)":
		return "Asia/Tokyo"
	case "Tokyo(Narita)":
		return "Asia/Tokyo"
	case "Vancouver":
		return "Canada/Pacific"
	case "Montreal":
		return "Canada/Eastern"
	default:
		panic("unrecognized location")
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	data, _ := ioutil.ReadAll(os.Stdin)
	lines := strings.Split(string(data), "\n")

	var flights []Flight

	chunkSize := 6

	for i := 0; i < len(lines); i += chunkSize {
		end := i + chunkSize

		if end > len(lines) {
			break
		}

		flight := Flight{
			FlightNumber:  lines[i],
			StartDateTime: lines[i+1],
			StartLoc:      lines[i+2],
			EndDateTime:   lines[i+3],
			EndLoc:        lines[i+4],
		}
		fmt.Printf("%v\n", flight)
		flights = append(flights, flight)
	}

	writeToGoogle(flights)
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func writeToGoogle(flights []Flight) {
	calID := os.Getenv("CAL_ID")

	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		log.Fatalf("Unable to retrieve token: %v", err)
	}

	client := config.Client(context.Background(), tok)

	srv, err := calendar.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	for _, flight := range flights {
		s, err := formattedDateTime(flight.StartDateTime, locationToZone(flight.StartLoc))
		if err != nil {
			log.Fatalf("Unable to format date time: %v", err)
		}
		e, err := formattedDateTime(flight.EndDateTime, locationToZone(flight.EndLoc))
		if err != nil {
			log.Fatalf("Unable to format date time: %v", err)
		}
		event := calendar.Event{
			Summary:     "[Auto] " + flight.FlightNumber,
			Description: flight.StartLoc + " > " + flight.EndLoc,
			Start:       &calendar.EventDateTime{DateTime: s.Format(time.RFC3339)},
			End:         &calendar.EventDateTime{DateTime: e.Format(time.RFC3339)},
		}

		res, err := srv.Events.Insert(calID, &event).Do()
		if err != nil {
			log.Fatalf("Unable to insert: %v", err)
		}
		fmt.Printf("Event created: %s\n", res.HtmlLink)
	}
}
