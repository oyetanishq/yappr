package model

import "time"

// User represents an authenticated GitHub user stored in MongoDB.
type User struct {
	ID        string    `bson:"_id"        json:"id"`
	GithubID  int64     `bson:"github_id"  json:"github_id"`
	Login     string    `bson:"login"      json:"login"`
	Name      string    `bson:"name"       json:"name"`
	Email     string    `bson:"email"      json:"email"`
	AvatarURL string    `bson:"avatar_url" json:"avatar_url"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}
