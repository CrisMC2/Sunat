package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/consulta-ruc-scraper/pkg/scraper"
)

func main() {
	rucs := []string{"20606316977"}

	if len(os.Args) > 1 {
		rucs = os.Args[1:]
	}

	sunatScraper, err := scraper.NewSUNATScraper()
	if err != nil {
		log.Fatal("Error creating scraper:", err)
	}
	defer sunatScraper.Close()

	for _, ruc := range rucs {
		fmt.Printf("Scraping RUC: %s\n", ruc)

		info, err := sunatScraper.ScrapeRUC(ruc)
		if err != nil {
			log.Printf("Error scraping RUC %s: %v\n", ruc, err)
			continue
		}

		jsonData, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			log.Printf("Error marshaling data for RUC %s: %v\n", ruc, err)
			continue
		}

		fmt.Println(string(jsonData))

		fileName := fmt.Sprintf("output/ruc_%s.json", ruc)
		err = os.WriteFile(fileName, jsonData, 0644)
		if err != nil {
			log.Printf("Error saving data for RUC %s: %v\n", ruc, err)
			continue
		}

		fmt.Printf("Data saved to %s\n\n", fileName)
	}
}
