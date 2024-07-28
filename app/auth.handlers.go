package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"time"
)

type contextKey string

const userContextKey = contextKey("user")

const cookieSessionName = "session"

var sessions = map[string]session{}

type session struct {
	username string
	userId   primitive.ObjectID
	expiry   time.Time
}

type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type UserContextData struct {
	ID   primitive.ObjectID
	Name string
}

func (s session) isExpired() bool {
	return s.expiry.Before(time.Now())
}

// SignInHandler godoc
// @Summary      Sign in
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request   body      main.Credentials  true  "Sign in credentials"
// @Router       /sign-in [post]
func SignInHandler(w http.ResponseWriter, r *http.Request) {
	var credentials Credentials
	err := json.NewDecoder(r.Body).Decode(&credentials)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	collection := mongoClient.Database(dbName).Collection(usersCollectionName)

	user, err := getUserByName(collection, credentials.Username)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	if user == nil || user.Password != credentials.Password {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	sessionToken, _ := generateSessionToken()
	expiresAt := time.Now().Add(60 * 60 * 10 * time.Second)

	sessions[sessionToken] = session{
		username: credentials.Username,
		userId:   user.ID,
		expiry:   expiresAt,
	}

	http.SetCookie(w, &http.Cookie{
		Name:    cookieSessionName,
		Value:   sessionToken,
		Expires: expiresAt,
	})

	w.Write([]byte("Signed in successfully"))
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(cookieSessionName)

	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// For any other type of error, return a bad request status
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sessionToken := cookie.Value

	delete(sessions, sessionToken)

	http.SetCookie(w, &http.Cookie{
		Name:    cookieSessionName,
		Value:   "",
		Expires: time.Now(),
	})
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieSessionName)
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		sessionToken := cookie.Value
		userSession, exists := sessions[sessionToken]
		if !exists || userSession.isExpired() {
			if exists {
				delete(sessions, sessionToken)
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userContext := &UserContextData{
			ID:   userSession.userId,
			Name: userSession.username,
		}

		ctx := context.WithValue(r.Context(), userContextKey, userContext)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func getUserByName(collection *mongo.Collection, name string) (*User, error) {
	var user User
	filter := bson.M{"name": name}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func generateRandomString(length int) (string, error) {
	tokenBytes := make([]byte, length)

	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}

	token := hex.EncodeToString(tokenBytes)

	return token, nil
}

func generateSessionToken() (string, error) {
	return generateRandomString(16)
}
