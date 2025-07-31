package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	DatabaseName   = "blog"
	CollectionName = "posts"
)

var Client *mongo.Client

// ConnectMongoDB 连接到 MongoDB
func ConnectMongoDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 连接字符串 - 使用 Docker 端口映射和认证
	clientOptions := options.Client().ApplyURI("mongodb://admin:password@localhost:27018")

	// 连接到 MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}

	// 测试连接
	err = client.Ping(ctx, nil)
	if err != nil {
		return err
	}

	Client = client
	log.Println("Successfully connected to MongoDB!")
	return nil
}

// GetCollection 获取集合
func GetCollection() *mongo.Collection {
	return Client.Database(DatabaseName).Collection(CollectionName)
}

// CloseMongoDB 关闭 MongoDB 连接
func CloseMongoDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return Client.Disconnect(ctx)
}
