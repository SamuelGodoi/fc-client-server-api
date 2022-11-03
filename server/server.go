package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.Println("START")
	db, err := sql.Open("sqlite3", "./cotacoes.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	err = CreateTable(db)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		cotacao, err := searchQuote(ctx)
		if err != nil {
			log.Println("Erro ao trazer a cotação", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*10)
		err = persisteNoBanco(ctx, db, cotacao)
		defer cancel()

		if err != nil {
			log.Println("Erro ao gravar cotacao no banco", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Erro ao consumir dados da API"))
			return
		}

		w.Header().Set("Content-Type", "applicatio/json")
		json.NewEncoder(w).Encode(struct {
			Bid float64 `json:"bid"`
		}{
			Bid: cotacao,
		})
	})
	http.ListenAndServe(":8080", nil)

}

func searchQuote(ctx context.Context) (float64, error) {
	log.Println("Procurando cotação da api")
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		log.Println(err)
		return 0.0, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return 0.0, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0.0, err
	}
	var data map[string]interface{}
	json.Unmarshal(body, &data)
	cotacao, _ := strconv.ParseFloat(data["USDBRL"].(map[string]interface{})["bid"].(string), 64)
	return cotacao, nil
}

func CreateTable(db *sql.DB) error {
	log.Println("Criando tabela caso não exista")
	sql := `
			CREATE TABLE IF NOT EXISTS cambio (
				data DATE, 
				cotacao REAL
			);
	`
	_, err := db.Exec(sql)
	return err
}

func persisteNoBanco(ctx context.Context, db *sql.DB, cotacao float64) error {
	log.Println("Tentando persitir cotação", cotacao)
	sql := `
			INSERT INTO cambio (data, cotacao)
			VALUES (?, ?)
	`
	stmt, err := db.Prepare(sql)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, time.Now(), cotacao)
	return err
}
