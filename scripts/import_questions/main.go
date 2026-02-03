package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Tehran",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	f, err := excelize.OpenFile("/Users/omid/Downloads/4A-Q.xlsx")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	sheets := f.GetSheetList()

	totalImported := 0

	for _, sheetName := range sheets {
		fmt.Printf("Importing sheet: %s\n", sheetName)
		rows, err := f.GetRows(sheetName)
		if err != nil {
			fmt.Printf("Error reading sheet %s: %v\n", sheetName, err)
			continue
		}

		for i, row := range rows {
			if i == 0 || len(row) < 7 { // Skip header or invalid rows
				continue
			}

			// row[0]: ID
			// row[1]: Question Text
			// row[2]: Opt1
			// row[3]: Opt2
			// row[4]: Opt3
			// row[5]: Opt4
			// row[6]: Correct Answer (Text like "گزینه ۱")

			questionText := row[1]
			options := []string{row[2], row[3], row[4], row[5]}

			correctAnswerIndicator := row[6]
			correctAnswerText := ""

			// Map "گزینه ۱" etc to the actual text
			switch {
			case strings.Contains(correctAnswerIndicator, "۱"):
				correctAnswerText = options[0]
			case strings.Contains(correctAnswerIndicator, "۲"):
				correctAnswerText = options[1]
			case strings.Contains(correctAnswerIndicator, "۳"):
				correctAnswerText = options[2]
			case strings.Contains(correctAnswerIndicator, "۴"):
				correctAnswerText = options[3]
			default:
				fmt.Printf("Invalid correct answer indicator: %s in row %d\n", correctAnswerIndicator, i)
				continue
			}

			optionsJSON, _ := json.Marshal(options)

			question := models.Question{
				QuestionText:  questionText,
				QuestionType:  models.QuestionTypeQuiz,
				Category:      sheetName,
				Difficulty:    models.DifficultyMedium,
				CorrectAnswer: correctAnswerText,
				Options:       string(optionsJSON),
				Points:        10,
			}

			if err := db.Create(&question).Error; err != nil {
				fmt.Printf("Error creating question in row %d: %v\n", i, err)
			} else {
				totalImported++
			}
		}
	}

	fmt.Printf("Successfully imported %d questions.\n", totalImported)
}
