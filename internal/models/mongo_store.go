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

// CreatePost 创建一篇新文章
func (s *MongoStore) CreatePost(post *Post) (primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 生成新的 ObjectID
	post.ID = primitive.NewObjectID()

	// 插入到数据库
	_, err := s.collection.InsertOne(ctx, post)
	if err != nil {
		return primitive.NilObjectID, err
	}

	log.Printf("Created post with ID: %s", post.ID.Hex())
	return post.ID, nil
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

// UpdatePost 更新文章
func (s *MongoStore) UpdatePost(id string, post *Post) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	post.UpdatedAt = time.Now()

	_, err = s.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{
			"title":      post.Title,
			"content":    post.Content,
			"updated_at": post.UpdatedAt,
		}},
	)

	return err
}

// DeletePost 删除文章
func (s *MongoStore) DeletePost(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = s.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}
