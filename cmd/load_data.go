package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/elastic/go-elasticsearch"
	"github.com/elastic/go-elasticsearch/esapi"
)

type Players struct {
	Players []NFLPlayers
}

type NFLPlayers struct {
	Name            string      `json:"player"`
	Team            string      `json:"team"`
	Pos             string      `json:"pos"`
	Att             float64     `json:"att"`
	AttPerGame      float64     `json:"att/g"`
	Yds             interface{} `json:"yds"`
	YdsClean        float64     `json:"ydsclean"`
	Avg             float64     `json:"avg"`
	YdsPerGame      float64     `json:"yds/g"`
	Td              float64     `json:"td"`
	Lng             string      `json:"lng"`
	LngClean        float64     `json:"lngclean"`
	First           float64     `json:"1st"`
	FirstPercentage float64     `json:"1st%"`
	TwentyPlus      float64     `json:"20+"`
	FourtyPlus      float64     `json:"40+"`
	FUM             float64     `json:"fum"`
}

// 1. Read Rushing.JSON file
// 2. Convert to Go Struct
// 3. Index documents concurrently
func main() {
	// wait group for goroutines
	var wg sync.WaitGroup
	// es client
	var es, _ = elasticsearch.NewDefaultClient()

	jsonFile, err := os.Open("rushing.json")
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var players []NFLPlayers
	json.Unmarshal(byteValue, &players)

	// Lng is needs to be cleaned because it contains non-numbered characters
	reg, err := regexp.Compile("[^0-9]+")

	for i, player := range players {
		wg.Add(1)
		// add documents concurrently
		go func(i int, p NFLPlayers) {
			defer wg.Done()
			if err != nil {
				log.Fatal(err)
			}

			// scrub Lng (string characters that are not numbers)
			p.Lng = reg.ReplaceAllString(p.Lng, "")
			p.LngClean, _ = strconv.ParseFloat(p.Lng, 64)

			// scrub yds (is either a string or number value)
			switch c := p.Yds.(type) {
			case string:
				p.YdsClean, _ = strconv.ParseFloat(c, 64)
			case float64:
				p.YdsClean = c
			default:
				fmt.Printf("garbage data, fix data pipelines...")
			}

			jsonString, _ := json.Marshal(p)
			request := esapi.IndexRequest{
				Index:      "nfl_players",
				DocumentID: strconv.Itoa(i + 1),
				Body:       strings.NewReader(string(jsonString)),
				Refresh:    "true",
			}
			request.Do(context.Background(), es)
		}(i, player)

	}
	wg.Wait()

	fmt.Println("complete")
}
