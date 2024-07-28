package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"slices"
	"strings"
	"time"
)

type CreteProfileRequestBody struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Avatar   string `json:"avatar"`
}

type UpdateProfileRequestBody struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type CreatePostRequestBody struct {
	Content string `json:"content"`
}

// CreateProfileHandler godoc
// @Summary      Create profile
// @Tags         profile
// @Accept       json
// @Produce      json
// @Param        request   body      main.CreteProfileRequestBody  true  "Create profile data"
// @Success      200  {object}  main.User
// @Router       /profile [post]
func CreateProfileHandler(w http.ResponseWriter, r *http.Request) {
	var createUserProfileData CreteProfileRequestBody
	err := json.NewDecoder(r.Body).Decode(&createUserProfileData)

	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var user = &User{
		ID:            primitive.NewObjectID(),
		Name:          createUserProfileData.Name,
		Avatar:        createUserProfileData.Avatar,
		Password:      createUserProfileData.Password,
		Posts:         []primitive.ObjectID{},
		LikedPosts:    []primitive.ObjectID{},
		Notifications: []primitive.ObjectID{},
	}

	collection := mongoClient.Database(dbName).Collection(usersCollectionName)

	_, err = getUserByName(collection, createUserProfileData.Name)
	if err == nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	_, err = collection.InsertOne(context.TODO(), user)
	if err != nil {
		http.Error(w, "Error creating createUserProfileData", http.StatusInternalServerError)
		return
	}

	sessionToken, _ := generateSessionToken()
	expiresAt := time.Now().Add(60 * 60 * 10 * time.Second)

	sessions[sessionToken] = session{
		username: user.Name,
		userId:   user.ID,
		expiry:   expiresAt,
	}

	http.SetCookie(w, &http.Cookie{
		Name:    cookieSessionName,
		Value:   sessionToken,
		Expires: expiresAt,
	})

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Account created successfully"))
}

// GetProfileHandler godoc
// @Summary      Get my profile
// @Tags         profile
// @Accept       json
// @Produce      json
// @Success      200  {object}  main.User
// @Router       /profile [get]
func GetProfileHandler(w http.ResponseWriter, r *http.Request) {
	userContextData := r.Context().Value(userContextKey).(*UserContextData)
	if userContextData == nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	userID := userContextData.ID

	var user User
	collection := mongoClient.Database(dbName).Collection(usersCollectionName)
	err := collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// UpdateProfileHandler godoc
// @Summary      Update my profile
// @Tags         profile
// @Accept       json
// @Produce      json
// @Param        request   body      main.UpdateProfileRequestBody  true  "Update profile data"
// @Success      200  {object}  main.User
// @Router       /profile [patch]
func UpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
	userContextData := r.Context().Value(userContextKey).(*UserContextData)
	if userContextData == nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	userID := userContextData.ID

	var updateProfileData UpdateProfileRequestBody
	err := json.NewDecoder(r.Body).Decode(&updateProfileData)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	updateFields := bson.M{}
	if updateProfileData.Name != "" {
		updateFields["name"] = updateProfileData.Name
	}
	if updateProfileData.Avatar != "" {
		updateFields["avatar"] = updateProfileData.Avatar
	}

	if len(updateFields) == 0 {
		http.Error(w, "No update fields provided", http.StatusBadRequest)
		return
	}

	userCollection := mongoClient.Database(dbName).Collection(usersCollectionName)
	_, err = userCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": userID},
		bson.M{"$set": updateFields},
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Profile updated successfully"))
}

// CreatePostHandler godoc
// @Summary      Create post
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param        request   body      main.CreatePostRequestBody  true  "Create post data"
// @Success      200  {object}  main.Post
// @Router       /posts [post]
func CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	userContextData := r.Context().Value(userContextKey).(*UserContextData)
	if userContextData == nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	userID := userContextData.ID

	var createPostData CreatePostRequestBody
	err := json.NewDecoder(r.Body).Decode(&createPostData)
	if err != nil {
		http.Error(w, "Failed to decode json", http.StatusBadRequest)
		return
	}

	post := &Post{
		ID:         primitive.NewObjectID(),
		Author:     userID,
		Content:    createPostData.Content,
		LikesCount: 0,
	}

	// TODO: add transaction
	collection := mongoClient.Database(dbName).Collection(postsCollectionName)
	_, err = collection.InsertOne(context.Background(), post)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userCollection := mongoClient.Database(dbName).Collection(usersCollectionName)
	_, err = userCollection.UpdateOne(context.Background(), bson.M{"_id": userID}, bson.M{"$addToSet": bson.M{"posts": post.ID}})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(post)
}

// GetMyPostsHandler godoc
// @Summary      Get my posts
// @Tags         posts
// @Accept       json
// @Produce      json
// @Success      200  {array}  main.Post
// @Router       /posts [get]
func GetMyPostsHandler(w http.ResponseWriter, r *http.Request) {
	userContextData := r.Context().Value(userContextKey).(*UserContextData)
	if userContextData == nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	userID := userContextData.ID

	var posts []Post
	collection := mongoClient.Database(dbName).Collection(postsCollectionName)
	cursor, err := collection.Find(context.Background(), bson.M{"author": userID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())
	for cursor.Next(context.Background()) {
		var post Post
		cursor.Decode(&post)
		posts = append(posts, post)
	}
	if err := cursor.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if posts == nil {
		posts = []Post{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

// GetLikedPostsHandler godoc
// @Summary      Get posts that I've liked
// @Tags         posts
// @Accept       json
// @Produce      json
// @Success      200  {array}  main.Post
// @Router       /posts/liked [get]
func GetLikedPostsHandler(w http.ResponseWriter, r *http.Request) {
	userContextData := r.Context().Value(userContextKey).(*UserContextData)
	if userContextData == nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	fmt.Print(userContextData)

	userID := userContextData.ID

	var user User
	collection := mongoClient.Database(dbName).Collection(usersCollectionName)

	err := collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	likedPosts := user.LikedPosts
	if likedPosts == nil {
		likedPosts = []primitive.ObjectID{}
	}

	var posts []Post
	postCollection := mongoClient.Database(dbName).Collection(postsCollectionName)
	cursor, err := postCollection.Find(context.Background(), bson.M{"_id": bson.M{"$in": likedPosts}})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())
	for cursor.Next(context.Background()) {
		var post Post
		cursor.Decode(&post)
		posts = append(posts, post)
	}
	if err := cursor.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if posts == nil {
		posts = []Post{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

// LikePostHandler godoc
// @Summary      Like post
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param     id   path      string  true  "ID of post to like"
// @Success      200  {object}  main.Post
// @Router       /posts/{id}/like [post]
func LikePostHandler(w http.ResponseWriter, r *http.Request) {
	urlPath := strings.TrimPrefix(r.URL.Path, "/posts/")
	urlPath = strings.TrimSuffix(urlPath, "/like")
	postIDFromQuery := urlPath

	userContextData := r.Context().Value(userContextKey).(*UserContextData)
	if userContextData == nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	userID := userContextData.ID
	postID, err := primitive.ObjectIDFromHex(postIDFromQuery)

	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	// TODO: wrap these updates into transaction

	var user User
	userCollection := mongoClient.Database(dbName).Collection(usersCollectionName)
	err = userCollection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)

	if slices.Contains(user.LikedPosts, postID) {
		http.Error(w, "Post is already liked by you", http.StatusInternalServerError)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	notification := Notification{
		ID:      primitive.NewObjectID(),
		Type:    "like",
		PostID:  postID,
		LikedBy: userID,
	}

	notificationCollection := mongoClient.Database(dbName).Collection(notificationsCollectionName)
	_, err = notificationCollection.InsertOne(context.Background(), notification)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	postsCollection := mongoClient.Database(dbName).Collection(postsCollectionName)
	_, err = postsCollection.UpdateOne(context.Background(), bson.M{"_id": postID}, bson.M{"$inc": bson.M{"likesCount": 1}})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = userCollection.UpdateOne(context.Background(), bson.M{"_id": userID}, bson.M{"$addToSet": bson.M{"likedPosts": postID}})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = userCollection.UpdateOne(context.Background(), bson.M{"_id": userID}, bson.M{"$addToSet": bson.M{"notifications": notification.ID}})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notification)
}

// GetNotificationsHandler godoc
// @Summary      Get notifications
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Success      200  {array}  main.Notification
// @Router       /notifications [get]
func GetNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	userContextData := r.Context().Value(userContextKey).(*UserContextData)
	if userContextData == nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	userID := userContextData.ID

	//var user User
	//collection := mongoClient.Database(dbName).Collection(usersCollectionName)
	//err := collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}

	var notifications []Notification
	notificationCollection := mongoClient.Database(dbName).Collection(notificationsCollectionName)
	cursor, err := notificationCollection.Find(context.Background(), bson.M{"likedBy": userID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())
	for cursor.Next(context.Background()) {
		var notification Notification
		cursor.Decode(&notification)
		notifications = append(notifications, notification)
	}
	if err := cursor.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if notifications == nil {
		notifications = []Notification{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}
