package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	serverAddress = "localhost:8080"
	dataFolder    = "data/"
	dbFilePath    = "quotation.db"
	dbTimeout     = 10 * time.Millisecond
	apiUrl        = "https://economia.awesomeapi.com.br/json/last/USD-BRL"
	apiTimeout    = 200 * time.Millisecond
)

type Quotation struct {
	Bid string `json:"bid"`
}

func main() {
	db, err := initDatabase()
	if err != nil {
		log.Fatal("Error initializing the database: ", err)
	}
	defer db.Close()

	log.Println("Starting server on:", serverAddress)
	http.HandleFunc("/cotacao", cotacaoHandler(db))
	if err := http.ListenAndServe(serverAddress, nil); err != nil {
		log.Fatal("Error starting server: ", err)
	}
}

func initDatabase() (*sql.DB, error) {
	if _, err := os.Stat(dataFolder); os.IsNotExist(err) {
		os.Mkdir(dataFolder, 0755)
	}

	db, err := sql.Open("sqlite3", dataFolder+dbFilePath)
	if err != nil {
		log.Fatalln("Error opening database:", err)
		return nil, err
	}
	log.Println("Database file created:", dataFolder+dbFilePath)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS bids (id INTEGER PRIMARY KEY AUTOINCREMENT, bid DOUBLE, created_at DATETIME)`)
	if err != nil {
		log.Fatalln("Error creating table:", err)
		return nil, err
	}

	return db, nil
}

func cotacaoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), apiTimeout)
		defer cancel()

		data, err := fetchDataFromApi(ctx)
		if err != nil {
			http.Error(w, "Error getting data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := saveData(db, data); err != nil {
			http.Error(w, "Error saving data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}

func fetchDataFromApi(ctx context.Context) (*Quotation, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var response map[string]Quotation
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Print(err)
		return nil, err
	}

	data, ok := response["USDBRL"]
	if !ok {
		return nil, fmt.Errorf("exchange rate not found")
	}
	log.Println(data.Bid)

	return &data, nil
}

func saveData(db *sql.DB, data *Quotation) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	floatValue, err := strconv.ParseFloat(data.Bid, 64)

	if err != nil {
		floatValue = 0.0
	}

	_, err = db.ExecContext(ctx, "INSERT INTO bids (bid, created_at) VALUES (?, ?)", floatValue, time.Now())
	return err
}
