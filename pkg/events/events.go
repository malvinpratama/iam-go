// Package events defines the asynchronous inter-service event contract and a
// thin NATS JetStream helper shared by the auth (producer) and user (consumer)
// services. Events are how the system stays a set of independent services:
// auth never calls the user service directly — it records an outbox row and a
// relay publishes it here; the user service consumes it idempotently.
package events

import (
	"time"

	"github.com/nats-io/nats.go"
)

// Stream and subjects. All user lifecycle events share the IAM_EVENTS stream.
const (
	StreamName            = "IAM_EVENTS"
	SubjectPrefix         = "iam."
	SubjectUserRegistered = "iam.user.registered"
	SubjectUserDeleted    = "iam.user.deleted"
)

// Event type tags as stored in the auth outbox (subject = SubjectPrefix + type).
const (
	TypeUserRegistered = "user.registered"
	TypeUserDeleted    = "user.deleted"
)

// UserRegistered is published after an identity is created. The user service
// uses it to create the matching profile.
type UserRegistered struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

// UserDeleted is published after an identity is deleted. The user service uses
// it to drop the matching profile.
type UserDeleted struct {
	UserID string `json:"user_id"`
}

// Connect opens a NATS connection (with infinite reconnect) and a JetStream
// context. Callers Close the returned *nats.Conn on shutdown.
func Connect(url string) (*nats.Conn, nats.JetStreamContext, error) {
	nc, err := nats.Connect(url,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return nil, nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, nil, err
	}
	return nc, js, nil
}

// EnsureStream creates the IAM_EVENTS stream if it does not already exist
// (idempotent — safe to call from every service on startup).
func EnsureStream(js nats.JetStreamContext) error {
	if _, err := js.StreamInfo(StreamName); err == nil {
		return nil
	}
	_, err := js.AddStream(&nats.StreamConfig{
		Name:      StreamName,
		Subjects:  []string{SubjectPrefix + "user.>"},
		Storage:   nats.FileStorage,
		Retention: nats.LimitsPolicy,
		MaxAge:    72 * time.Hour,
	})
	return err
}
