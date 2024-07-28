package main

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID            primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Name          string               `bson:"name" json:"name"`
	Password      string               `bson:"password" json:"password"`
	Avatar        string               `bson:"avatar" json:"avatar"`
	Posts         []primitive.ObjectID `bson:"posts" json:"posts"`
	LikedPosts    []primitive.ObjectID `bson:"likedPosts" json:"likedPosts"`
	Notifications []primitive.ObjectID `bson:"notifications" json:"notifications"`
}

type Post struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Content    string             `bson:"content" json:"content"`
	Author     primitive.ObjectID `bson:"author" json:"author"`
	LikesCount int                `bson:"likesCount" json:"likesCount"`
}

type Notification struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type    string             `bson:"type" json:"type"`
	PostID  primitive.ObjectID `bson:"postId" json:"postId"`
	LikedBy primitive.ObjectID `bson:"likedBy" json:"likedBy"`
}
