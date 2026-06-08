// Package grpcutil carries the authenticated identity between the gateway and
// internal services via gRPC metadata. Internal services trust this metadata
// because only the gateway (on the internal network) sets it.
package grpcutil

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	MDUserID        = "x-user-id"
	MDEmail         = "x-user-email"
	MDRoles         = "x-user-roles"       // comma-separated
	MDPermissions   = "x-user-permissions" // comma-separated
	MDInternalToken = "x-internal-token"   // gateway→service shared secret
)

// HasPermission reports whether the identity holds the given permission.
func (id Identity) HasPermission(perm string) bool {
	for _, p := range id.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// Identity is the caller identity resolved by the gateway.
type Identity struct {
	UserID      string
	Email       string
	Roles       []string
	Permissions []string
}

// Inject attaches the identity to an outgoing context.
func Inject(ctx context.Context, id Identity) context.Context {
	md := metadata.Pairs(
		MDUserID, id.UserID,
		MDEmail, id.Email,
		MDRoles, strings.Join(id.Roles, ","),
		MDPermissions, strings.Join(id.Permissions, ","),
	)
	return metadata.NewOutgoingContext(ctx, md)
}

// FromIncoming extracts the identity from an incoming server context.
func FromIncoming(ctx context.Context) Identity {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return Identity{}
	}
	id := Identity{
		UserID: first(md, MDUserID),
		Email:  first(md, MDEmail),
	}
	if roles := first(md, MDRoles); roles != "" {
		id.Roles = strings.Split(roles, ",")
	}
	if perms := first(md, MDPermissions); perms != "" {
		id.Permissions = strings.Split(perms, ",")
	}
	return id
}

func first(md metadata.MD, key string) string {
	if v := md.Get(key); len(v) > 0 {
		return v[0]
	}
	return ""
}
