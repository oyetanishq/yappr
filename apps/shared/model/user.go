package model

import "time"

// Plan is the subscription tier a user is on.
type Plan string

const (
	PlanFree Plan = "free"
	PlanPro  Plan = "pro"
)

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

	// Billing / subscription fields
	Plan                   Plan       `bson:"plan"                      json:"plan"`
	RazorpaySubscriptionID string     `bson:"razorpay_subscription_id"  json:"razorpay_subscription_id,omitempty"`
	PlanExpiresAt          *time.Time `bson:"plan_expires_at"           json:"plan_expires_at,omitempty"`
	PRCountThisMonth       int        `bson:"pr_count_this_month"       json:"pr_count_this_month"`
	PRCountResetAt         time.Time  `bson:"pr_count_reset_at"         json:"pr_count_reset_at"`
}

// IsPro reports whether the user has an active Pro subscription.
func (u *User) IsPro() bool {
	if u.Plan != PlanPro {
		return false
	}
	if u.PlanExpiresAt == nil {
		return false
	}
	return time.Now().Before(*u.PlanExpiresAt)
}

// PRLimitReached reports whether a free-tier user has hit the monthly PR review cap.
const FreePRLimit = 10

func (u *User) PRLimitReached() bool {
	if u.IsPro() {
		return false
	}
	// Reset counter if we're in a new calendar month.
	now := time.Now().UTC()
	if now.Year() != u.PRCountResetAt.Year() || now.Month() != u.PRCountResetAt.Month() {
		return false // will be reset on next increment
	}
	return u.PRCountThisMonth >= FreePRLimit
}
