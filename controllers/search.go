package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"nfl-rushing/models"
	"nfl-rushing/restapi/operations"

	"github.com/olivere/elastic/v7"
)

func SearchHandler(params operations.SearchParams) (response *models.SearchResponseBody, err error) {

	// init es client
	esClient, err := elastic.NewClient(elastic.SetURL("http://localhost:9200"),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false))
	if err != nil {
		fmt.Println("Error initializing : ", err)
		return nil, err
	}

	fmt.Println("ES initialized...")

	var matchQuery *elastic.MatchQuery
	var matchAllQuery *elastic.MatchAllQuery
	var sort *elastic.FieldSort

	esSource := elastic.NewSearchSource()

	// add filter by player name; defaults to all players if not specified
	if params.PlayerName != nil {
		matchQuery = elastic.NewMatchQuery("player", *params.PlayerName)
		esSource.Query(matchQuery)
	} else {
		matchAllQuery = (*elastic.MatchAllQuery)(elastic.NewMatchAllQuery())
		esSource.Query(matchAllQuery)
	}

	// add sort fields for td, yds and lng
	if params.Sort != nil {
		sortField := *params.Sort
		if sortField == "td" {
			sort = elastic.NewFieldSort(sortField).Desc()
			esSource.SortBy(sort)
		}
		// Lng is in string so need to be sep logic
		if sortField == "lng" {
			sort = elastic.NewFieldSort("lngclean").Desc()
			esSource.SortBy(sort)
		}
		// Lng is in string so need to be sep logic
		if sortField == "yds" {
			sort = elastic.NewFieldSort("ydsclean").Desc()
			esSource.SortBy(sort)
		}
	}

	// buid es search source with index, query and sort fields
	searchService := esClient.Search().Index("nfl_players").SearchSource(esSource).Size(1000)

	// send es search request
	ctx := context.Background()
	searchResult, err := searchService.Do(ctx)
	if err != nil {
		fmt.Println("Error performing search request...")
		return nil, err
	}

	// es results to search response
	var players []*models.NFLPlayer
	for _, hit := range searchResult.Hits.Hits {
		var p *models.NFLPlayer
		// ignore error handling for now
		json.Unmarshal(hit.Source, &p)
		players = append(players, p)
	}

	response = &models.SearchResponseBody{NflPlayers: players}

	return response, err
}
