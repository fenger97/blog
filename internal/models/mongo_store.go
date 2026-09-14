package models

import (
	"context"
	"log"
	"time"

	"blog/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoStore MongoDB 存储实现
type MongoStore struct {
	collection *mongo.Collection
}

// NewMongoStore 创建新的 MongoDB 存储实例
func NewMongoStore() *MongoStore {
	return &MongoStore{
		collection: database.GetCollection(),
	}
}

// GetPost 获取指定ID的文章
func (s *MongoStore) GetPost(id string) (*Post, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var post Post
	err = s.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&post)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // 文章不存在
		}
		return nil, err
	}

	return &post, nil
}

// GetAllPosts 获取所有文章，按创建时间倒序排列
func (s *MongoStore) GetAllPosts() ([]*Post, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Fetching all posts from MongoDB...")

	// 按创建时间倒序排列
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := s.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		log.Printf("Error finding posts: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var posts []*Post
	if err = cursor.All(ctx, &posts); err != nil {
		log.Printf("Error decoding posts: %v", err)
		return nil, err
	}

	log.Printf("Found %d posts", len(posts))
	return posts, nil
}
