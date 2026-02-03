package main

import (
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"
)

func main() {
	f, err := excelize.OpenFile("/Users/omid/Downloads/4A-Q.xlsx")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Get all the sheet names
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		log.Fatal("no sheets found")
	}

	fmt.Printf("Sheets: %v\n", sheets)

	// Read first sheet
	sheetName := sheets[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Fatal(err)
	}

	for i, row := range rows {
		if i > 5 {
			break
		}
		fmt.Printf("Row %d: %v\n", i, row)
	}
}
