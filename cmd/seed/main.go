package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var chaiNames = []struct{ bn, en string }{
	{"মিল্ক টি", "Milk Tea"}, {"লেবু চা", "Lemon Tea"}, {"আদা চা", "Ginger Tea"}, {"মশলা চা", "Masala Tea"},
	{"গ্রিন টি", "Green Tea"}, {"ব্ল্যাক কফি", "Black Coffee"}, {"ক্যাপুচিনো", "Cappuccino"}, {"লাটে", "Latte"},
	{"সিঙ্গারা", "Singara"}, {"সমুচা", "Samosa"}, {"পরোটা", "Paratha"}, {"খিচুড়ি", "Khichuri"},
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

		// এক statement এ item আর দুটো translation — আধা-আধি insert হবে না
		_, err := pool.Exec(
			context.Background(),
			`WITH item AS (INSERT INTO menu_items (price) VALUES ($1) RETURNING id)
			 INSERT INTO menu_item_translations (item_id, lang, name)
			 SELECT item.id, t.lang, t.name FROM item, (VALUES ('bn', $2::text), ('en', $3::text)) AS t(lang, name)`,
			price,
			fmt.Sprintf("%s #%d", name.bn, i+1),
			fmt.Sprintf("%s #%d", name.en, i+1),
		)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	fmt.Println("✅ 1000টা item ঢোকানো হয়েছে!")
}
