package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {

	// 실행 시 batch 번호를 인자로 받기
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <batch-number>", os.Args[0])
	}
	batchNumber, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid batch number: %s", os.Args[1])
	}
	log.Printf("Setting active tokens for batch number: %d", batchNumber)

	// MongoDB URI 설정
	databaseUri := os.Getenv("DATABASE_URI")
	if databaseUri == "" {
		databaseUri = "mongodb://localhost:27017"
		log.Printf("No DATABASE_URI environment variable found, using default value %s", databaseUri)
	} else {
		log.Printf("DATABASE_URI: %s", databaseUri)
	}

	// MongoDB 클라이언트 설정
	clientOptions := options.Client().ApplyURI(databaseUri).SetRetryWrites(true)
	dbClient, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer dbClient.Disconnect(context.Background())

	// `tokens` 컬렉션 설정
	collection := dbClient.Database("twitter").Collection("tokens")

	// MongoDB 세션 시작
	session, err := dbClient.StartSession()
	if err != nil {
		log.Fatalf("Failed to start MongoDB session: %v", err)
	}
	defer session.EndSession(context.Background())

	// 트랜잭션을 실행
	_, err = session.WithTransaction(context.Background(), func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Step 1: `batch` 필드가 없는 문서의 `batch` 값을 0으로 초기화
		_, err := collection.UpdateMany(
			sessCtx,
			bson.M{"batch": bson.M{"$exists": false}},
			bson.M{"$set": bson.M{"batch": 0}},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to set batch=0 for missing batch field: %v", err)
		}

		// Step 2: `batch`가 batchNumber인 문서의 `active` 상태를 `true`로 설정
		_, err = collection.UpdateMany(
			sessCtx,
			bson.M{"batch": batchNumber},
			bson.M{"$set": bson.M{"active": true}},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to set active=true for batch=2: %v", err)
		}

		// Step 3: `batch`가 batchNumber가 아닌 모든 문서의 `active` 상태를 `false`로 설정
		_, err = collection.UpdateMany(
			sessCtx,
			bson.M{"batch": bson.M{"$ne": batchNumber}},
			bson.M{"$set": bson.M{"active": false}},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to set active=false for batch!=2: %v", err)
		}

		// Step 4: `active` 상태가 `true`인 모든 문서의 `username` 필드를 로그로 출력
		cursor, err := collection.Find(sessCtx, bson.M{"active": true})
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve active tokens: %v", err)
		}
		defer cursor.Close(sessCtx)

		var activeUsers []string
		for cursor.Next(sessCtx) {
			var result bson.M
			if err := cursor.Decode(&result); err != nil {
				log.Printf("Failed to decode document: %v", err)
				continue
			}
			username, ok := result["username"].(string)
			if ok {
				activeUsers = append(activeUsers, username)
			}
		}

		if len(activeUsers) > 0 {
			log.Printf("Active Tokens Usernames: %v", activeUsers)
		} else {
			log.Println("No active tokens found.")
		}

		// 트랜잭션 완료
		return nil, nil
	})

	if err != nil {
		log.Fatalf("Transaction failed: %v", err)
	} else {
		log.Println("Transaction completed successfully.")
	}
}
