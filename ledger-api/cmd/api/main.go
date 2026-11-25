package main

import (
	"context"
	"crypto/subtle"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"ledger-api/internal/middleware"
	"ledger-api/internal/module/raw_transactions"
)

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing env var %s", k)
	}
	return v
}

type user struct{ Name, IngestSecret, MongoURI, DB string }

func main() {
	_ = godotenv.Load(".env") // local only; Lambda uses real env vars
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Each person's SMS lands in their own cluster and database: the ingest key they send picks which one.
	// A person is on when <NAME>_SECRET is set, and then needs <NAME>_MONGODB_URI and <NAME>_MONGODB_DB too. Names come from PEOPLE (comma-separated, default ME).
	var users []user
	for _, p := range people() {
		if os.Getenv(p+"_SECRET") != "" {
			users = append(users, user{strings.ToLower(p), os.Getenv(p + "_SECRET"), mustEnv(p + "_MONGODB_URI"), mustEnv(p + "_MONGODB_DB")})
		}
	}
	if len(users) == 0 {
		log.Fatal("set <NAME>_SECRET (default name ME) with its MONGODB_URI / MONGODB_DB; see .env.example")
	}
	type tenant struct {
		key string
		h   http.Handler
	}
	var ts []tenant
	seenName, seenKey := map[string]bool{}, map[string]bool{}
	for _, u := range users {
		if u.Name == "" || u.IngestSecret == "" || u.MongoURI == "" || u.DB == "" || seenName[u.Name] || seenKey[u.IngestSecret] {
			log.Fatalf("%q needs its own secret, mongodb_uri and db (two people cannot share a secret)", u.Name)
		}
		seenName[u.Name], seenKey[u.IngestSecret] = true, true
		client, err := mongo.Connect(options.Client().ApplyURI(u.MongoURI))
		if err != nil {
			log.Fatalf("%s: %v", u.Name, err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = client.Ping(ctx, nil)
		cancel()
		if err != nil { // not fatal: one person's cluster being down must not stop everyone else's ingest; their requests answer 500 until it is back
			log.Printf("mongo ping (%s) failed, continuing: %v", u.Name, err)
		}
		m := http.NewServeMux()
		raw_transactions.RegisterRoutes(m, raw_transactions.NewController(raw_transactions.NewService(client.Database(u.DB), u.Name)), middleware.APIKey(u.IngestSecret))
		ts = append(ts, tenant{u.IngestSecret, m})
	}
	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("X-API-Key"))
		var hit http.Handler
		for _, t := range ts { // no early exit, so timing does not reveal which key was close
			if subtle.ConstantTimeCompare(got, []byte(t.key)) == 1 {
				hit = t.h
			}
		}
		if hit == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		hit.ServeHTTP(w, r)
	})

	// Client above is created once per cold start and reused by warm invocations.
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(httpadapter.NewV2(mux).ProxyWithContext)
		return
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// people is the list of name prefixes from PEOPLE (default "ME"), upper-cased.
func people() []string {
	list := os.Getenv("PEOPLE")
	if list == "" {
		list = "ME"
	}
	var out []string
	for _, n := range strings.Split(list, ",") {
		if n = strings.ToUpper(strings.TrimSpace(n)); n != "" {
			out = append(out, n)
		}
	}
	return out
}
