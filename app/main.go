package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"
	"strings"
	"time"
)

var mongoClient *mongo.Client

const (
	mongoURL                    = "mongodb://user:pass@localhost:27017"
	port                        = 8085
	dbName                      = "social-network"
	postsCollectionName         = "posts"
	usersCollectionName         = "users"
	notificationsCollectionName = "notifications"
)

func methodHandler(method string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, fmt.Sprintf("Method not allowed. Method: %s Path: %s", r.Method, r.URL.Path), http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	}

}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(mongoURL)

	var err error
	mongoClient, err = mongo.Connect(ctx, clientOptions)

	if err != nil {
		log.Fatal(err)
	}

	log.Println(">>> Connected to mongodb")

	http.HandleFunc("/sign-in", methodHandler(http.MethodPost, SignInHandler))
	http.HandleFunc("/logout", methodHandler(http.MethodPost, LogoutHandler))

	http.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			methodHandler(http.MethodPost, CreateProfileHandler)(w, r)
		} else if r.Method == http.MethodGet {
			authMiddleware(methodHandler(http.MethodGet, GetProfileHandler))(w, r)
		} else if r.Method == http.MethodPatch {
			authMiddleware(methodHandler(http.MethodPatch, UpdateProfileHandler))(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			authMiddleware(methodHandler(http.MethodPost, CreatePostHandler))(w, r)
		} else if r.Method == http.MethodGet {
			authMiddleware(methodHandler(http.MethodGet, GetMyPostsHandler))(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/posts/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Match /posts/:id/like
			if strings.HasSuffix(r.URL.Path, "/like") {
				authMiddleware(methodHandler(http.MethodPost, LikePostHandler))(w, r)
				return
			}
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	http.HandleFunc("/posts/liked", authMiddleware(methodHandler(http.MethodGet, GetLikedPostsHandler)))
	http.HandleFunc("/notifications", authMiddleware(methodHandler(http.MethodGet, GetNotificationsHandler)))

	log.Printf(">>> Starting server on port %d...\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))

}
