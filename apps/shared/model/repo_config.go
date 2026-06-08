package model

import "time"

// Personality defines the tone/style the AI reviewer uses when writing comments.
type Personality string

const (
	// PersonalityBestie is casual, supportive, emoji-heavy, Gen-Z friendly.
	PersonalityBestie Personality = "bestie"
	// PersonalitySeniorDev is professional, precise, and mentorship-focused (default).
	PersonalitySeniorDev Personality = "senior_dev"
	// PersonalitySigma is extremely terse, no fluff, bullet-only, no pleasantries.
	PersonalitySigma Personality = "sigma"
	// PersonalityToxicTechLead is brutally critical and sarcastic — but technically accurate.
	PersonalityToxicTechLead Personality = "toxic_tech_lead"
)

// DefaultPersonality is used when no personality has been configured for a repo.
const DefaultPersonality = PersonalitySeniorDev

// IsValid reports whether p is one of the four recognised personalities.
func (p Personality) IsValid() bool {
	switch p {
	case PersonalityBestie, PersonalitySeniorDev, PersonalitySigma, PersonalityToxicTechLead:
		return true
	}
	return false
}

// RepoConfig stores per-repository Yappr configuration.
// It is keyed by repo_full_name ("owner/repo") and scoped to a user.
type RepoConfig struct {
	ID           string      `bson:"_id"            json:"id"`
	RepoFullName string      `bson:"repo_full_name" json:"repo_full_name"` // "owner/repo"
	UserID       string      `bson:"user_id"        json:"user_id"`
	IgnoredPaths []string    `bson:"ignored_paths"  json:"ignored_paths"` // globs/paths, one per entry
	Personality  Personality `bson:"personality"    json:"personality"`
	CreatedAt    time.Time   `bson:"created_at"     json:"created_at"`
	UpdatedAt    time.Time   `bson:"updated_at"     json:"updated_at"`
}
