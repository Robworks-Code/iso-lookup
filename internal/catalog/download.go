package catalog

import (
	"fmt"
	"io"
	"net/http"
)

const openDataBase = "https://isopublicstorageprod.blob.core.windows.net/opendata/_latest"

// URLs for the three Open Data sources.
var (
	DeliverablesURL = openDataBase + "/iso_deliverables_metadata/json/iso_deliverables_metadata.jsonl"
	CommitteesURL   = openDataBase + "/iso_technical_committees/json/iso_technical_committees.jsonl"
	ICSURL          = openDataBase + "/iso_ics/csv/ICS.csv"
)

func fetchURL(client *http.Client, url string) (io.ReadCloser, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	return resp.Body, nil
}

// BuildIndex downloads all three datasets and returns ingested Records.
func BuildIndex(client *http.Client) ([]Record, error) {
	del, err := fetchURL(client, DeliverablesURL)
	if err != nil {
		return nil, err
	}
	defer del.Close()
	com, err := fetchURL(client, CommitteesURL)
	if err != nil {
		return nil, err
	}
	defer com.Close()
	ics, err := fetchURL(client, ICSURL)
	if err != nil {
		return nil, err
	}
	defer ics.Close()
	return Ingest(del, com, ics)
}
