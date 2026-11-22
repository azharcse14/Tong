package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var chaiNames = []string{
	"মিল্ক টি", "লেবু চা", "আদা চা", "মশলা চা",
	"গ্রিন টি", "ব্ল্যাক কফি", "ক্যাপুচিনো", "লাটে",
	"সিঙ্গারা", "সমুচা", "পরোটা", "খিচুড়ি",
}

func main() {
	godotenv.Load()

	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	if err != nil {
		fmt.Println("DB connect হয়নি:", err)
		return
	}
	defer pool.Close()

	for i := 0; i < 1000; i++ {
		name := chaiNames[rand.Intn(len(chaiNames))]
		price := float64(rand.Intn(90)+10) // 10 থেকে 100 টাকা

		_, err := pool.Exec(
			context.Background(),
			"INSERT INTO menu_items (name, price) VALUES ($1, $2)",
			fmt.Sprintf("%s #%d", name, i+1), price,
		)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	fmt.Println("✅ 1000টা item ঢোকানো হয়েছে!")
}