package model

import "time"

// Session represents a persistent active session stored in MongoDB.
// Redis holds the same session as a fast lookup cache; MongoDB is the
// source of truth for listing and cross-device revocation.
type Session struct {
	ID        string    `bson:"_id"        json:"id"`
	UserID    string    `bson:"user_id"    json:"-"`
	UserAgent string    `bson:"user_agent" json:"user_agent"`
	IP        string    `bson:"ip"         json:"ip"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	ExpiresAt time.Time `bson:"expires_at" json:"expires_at"`
}
