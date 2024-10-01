package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var dbClient *mongo.Client

func init() {
	// MongoDB URI 설정
	databaseUri := os.Getenv("DATABASE_URI")
	if databaseUri == "" {
		databaseUri = "mongodb://localhost:27017"
		log.Printf("No DATABASE_URI environment variable found, using default value %s", databaseUri)
	} else {
		log.Printf("DATABASE_URI: %.10s...", databaseUri)
	}

	// MongoDB 클라이언트 설정
	clientOptions := options.Client().ApplyURI(databaseUri).SetRetryWrites(true)
	var err error
	dbClient, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
}

func rotate(args int) {

	batchNumber := args

	if batchNumber < 1 {
		log.Fatalf("Invalid batch number: %d", batchNumber)
	}

	log.Printf("⏰ Setting active tokens for batch number: %d", batchNumber)

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
			log.Printf("📃 Active Tokens Usernames: %v", activeUsers)
		} else {
			log.Println("Error no active tokens found.")
		}

		// 트랜잭션 완료
		return nil, nil
	})

	if err != nil {
		log.Fatalf("❗️ Error transaction failed: %v", err)
	} else {
		log.Println("🎉 Transaction completed successfully.")
	}
}

type Token struct {
	ObjectID primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Auth     string             `bson:"auth,omitempty" json:"auth,omitempty"`
	Ct0      string             `bson:"ct0,omitempty" json:"ct0,omitempty"`
	Username string             `bson:"username,omitempty" json:"username,omitempty"`
	Batch    int                `bson:"batch,omitempty" json:"batch,omitempty"`
	Active   bool               `bson:"active,omitempty" json:"active,omitempty"`
}

func main() {
	// 프로그램 종료 신호 수신 (CTRL+C 등)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go scheduleRotations() // 주기적 작업 스케줄러 시작

	log.Println("😄 Rotate service is running... Press Ctrl+C to exit.")
	<-stop // 프로그램이 종료될 때까지 대기

	log.Println("Shutting down...")
}

// 특정 시간 간격마다 작업을 실행하도록 하는 함수
func scheduleRotations() {
	ticker := time.NewTicker(8 * time.Second)
	defer ticker.Stop()

	// for t := range ticker.C {
	// 	jobID := int((t.Unix()/10)%3 + 1) // 1, 2, 3 중 하나의 작업 ID 선택
	// 	log.Printf("🔔 Triggering rotation job %d at %v\n", jobID, t)
	// 	rotate(jobID)
	// }

	for range ticker.C {

		// 현재 active: true인 batch 번호를 database에서 가져옴
		var token *Token
		collection := dbClient.Database("twitter").Collection("tokens")
		err := collection.FindOne(context.Background(), bson.M{"active": true}).Decode(&token)

		if err != nil {
			log.Fatalf("Failed to get current batch number: %v", err)
		}

		batchNumber := token.Batch

		// check null
		if batchNumber == 0 {
			log.Printf("❌ Failed to get current batch number: %v", err)
		}

		if batchNumber < 1 {
			log.Printf("❌ Invalid batch number: %d", batchNumber)
		}

		log.Printf("📃 Current batch number: %d", batchNumber)

		// 현재의 batch 번호에서 1 증가시키고 3을 초과하면 1로 돌아감
		batchNumber = (batchNumber%3 + 1)

		// 새로운 batch 번호로 rotate 함수 실행
		rotate(batchNumber)
	}

}
