// Package handler implements the AuthService gRPC server.
package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/malvin/iam-go/gen/auth/v1"
	"github.com/malvin/iam-go/pkg/grpcutil"
	"github.com/malvin/iam-go/pkg/jwt"
	"github.com/malvin/iam-go/pkg/password"
	"github.com/malvin/iam-go/services/auth/internal/db"
)

const defaultRole = "user"

// AuthHandler implements authv1.AuthServiceServer.
type AuthHandler struct {
	authv1.UnimplementedAuthServiceServer
	pool       *pgxpool.Pool
	q          *db.Queries
	jwt        *jwt.Manager
	refreshTTL time.Duration
	dummyHash  string // for constant-time login on unknown users
}

// New builds an AuthHandler.
func New(pool *pgxpool.Pool, jwtMgr *jwt.Manager, refreshTTL time.Duration) *AuthHandler {
	// Precompute an argon2 hash so Login spends comparable time whether or not
	// the user exists (mitigates user-enumeration via timing).
	dummy, _ := password.Hash("constant-time-dummy-password")
	return &AuthHandler{pool: pool, q: db.New(pool), jwt: jwtMgr, refreshTTL: refreshTTL, dummyHash: dummy}
}

// requirePerm enforces a permission from the gateway-supplied identity metadata
// (defense-in-depth: services re-check, not just the gateway).
func requirePerm(ctx context.Context, perm string) error {
	if grpcutil.FromIncoming(ctx).HasPermission(perm) {
		return nil
	}
	return status.Error(codes.PermissionDenied, "permission denied: "+perm)
}

// Register creates a user, assigns the default role, and returns the user id.
func (h *AuthHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}
	hash, err := password.Hash(req.GetPassword())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	// Create user + assign default role atomically.
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "tx begin failed")
	}
	defer tx.Rollback(ctx)
	qtx := h.q.WithTx(tx)

	user, err := qtx.CreateUser(ctx, db.CreateUserParams{Email: req.GetEmail(), PasswordHash: hash})
	if err != nil {
		return nil, status.Error(codes.AlreadyExists, "email already registered")
	}
	if err := qtx.AssignRoleToUser(ctx, db.AssignRoleToUserParams{UserID: user.ID, Name: defaultRole}); err != nil {
		return nil, status.Error(codes.Internal, "failed to assign default role")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "tx commit failed")
	}

	return &authv1.RegisterResponse{UserId: user.ID.String(), Email: user.Email}, nil
}

// Login verifies credentials and issues an access + refresh token pair.
func (h *AuthHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.TokenPair, error) {
	user, err := h.q.GetUserByEmail(ctx, req.GetEmail())
	if err != nil {
		// Unknown user: still run a hash compare so timing doesn't leak existence.
		password.Verify(h.dummyHash, req.GetPassword())
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if !password.Verify(user.PasswordHash, req.GetPassword()) {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	return h.issueTokens(ctx, user.ID, user.Email)
}

// Refresh rotates a valid refresh token for a new token pair.
func (h *AuthHandler) Refresh(ctx context.Context, req *authv1.RefreshRequest) (*authv1.TokenPair, error) {
	hash := hashToken(req.GetRefreshToken())
	row, err := h.q.GetRefreshToken(ctx, hash)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}
	if row.RevokedAt.Valid {
		return nil, status.Error(codes.Unauthenticated, "refresh token revoked")
	}
	if row.ExpiresAt.Valid && row.ExpiresAt.Time.Before(time.Now()) {
		return nil, status.Error(codes.Unauthenticated, "refresh token expired")
	}
	user, err := h.q.GetUserByID(ctx, row.UserID)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "user not found")
	}
	// Rotate: revoke the presented token, then issue a fresh pair.
	if err := h.q.RevokeRefreshToken(ctx, hash); err != nil {
		return nil, status.Error(codes.Internal, "failed to rotate token")
	}
	return h.issueTokens(ctx, user.ID, user.Email)
}

// Logout revokes the refresh token and denylists the access token (by jti).
func (h *AuthHandler) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if err := h.q.RevokeRefreshToken(ctx, hashToken(req.GetRefreshToken())); err != nil {
		return nil, status.Error(codes.Internal, "failed to revoke token")
	}
	// Best-effort: denylist the access token so it stops working immediately.
	if at := req.GetAccessToken(); at != "" {
		if claims, err := h.jwt.Parse(at); err == nil && claims.ID != "" && claims.ExpiresAt != nil {
			_ = h.q.RevokeAccessJTI(ctx, db.RevokeAccessJTIParams{
				Jti:       claims.ID,
				ExpiresAt: pgtype.Timestamptz{Time: claims.ExpiresAt.Time, Valid: true},
			})
		}
	}
	return &authv1.LogoutResponse{Success: true}, nil
}

// ValidateToken verifies an access token and returns the caller's roles + permissions.
func (h *AuthHandler) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	claims, err := h.jwt.Parse(req.GetAccessToken())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
	}
	if claims.ID != "" {
		revoked, err := h.q.IsTokenRevoked(ctx, claims.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to check token status")
		}
		if revoked {
			return nil, status.Error(codes.Unauthenticated, "token revoked")
		}
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid subject")
	}
	roles, err := h.q.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load roles")
	}
	perms, err := h.q.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load permissions")
	}
	return &authv1.ValidateTokenResponse{
		UserId:      claims.Subject,
		Email:       claims.Email,
		Roles:       roles,
		Permissions: perms,
	}, nil
}

// DeleteUser removes the identity entirely (FK cascade drops roles & refresh
// tokens). Requires user:delete.
func (h *AuthHandler) DeleteUser(ctx context.Context, req *authv1.DeleteUserRequest) (*authv1.DeleteUserResponse, error) {
	if err := requirePerm(ctx, "user:delete"); err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	if err := h.q.DeleteUser(ctx, userID); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete user")
	}
	return &authv1.DeleteUserResponse{Success: true}, nil
}

// ── RBAC management ─────────────────────────────────────────

func (h *AuthHandler) CreateRole(ctx context.Context, req *authv1.CreateRoleRequest) (*authv1.Role, error) {
	if err := requirePerm(ctx, "role:write"); err != nil {
		return nil, err
	}
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "role name is required")
	}
	role, err := h.q.CreateRole(ctx, db.CreateRoleParams{Name: req.GetName(), Description: req.GetDescription()})
	if err != nil {
		return nil, status.Error(codes.AlreadyExists, "role already exists")
	}
	return &authv1.Role{Id: role.ID, Name: role.Name, Description: role.Description}, nil
}

func (h *AuthHandler) UpdateRole(ctx context.Context, req *authv1.UpdateRoleRequest) (*authv1.Role, error) {
	if err := requirePerm(ctx, "role:write"); err != nil {
		return nil, err
	}
	role, err := h.q.UpdateRole(ctx, db.UpdateRoleParams{Name: req.GetName(), Description: req.GetDescription()})
	if err != nil {
		return nil, status.Error(codes.NotFound, "role not found")
	}
	return &authv1.Role{Id: role.ID, Name: role.Name, Description: role.Description}, nil
}

func (h *AuthHandler) DeleteRole(ctx context.Context, req *authv1.DeleteRoleRequest) (*authv1.DeleteRoleResponse, error) {
	if err := requirePerm(ctx, "role:write"); err != nil {
		return nil, err
	}
	if isBuiltinRole(req.GetName()) {
		return nil, status.Error(codes.FailedPrecondition, "cannot delete built-in role")
	}
	if _, err := h.q.GetRoleByName(ctx, req.GetName()); err != nil {
		return nil, status.Error(codes.NotFound, "role not found")
	}
	if err := h.q.DeleteRole(ctx, req.GetName()); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete role")
	}
	return &authv1.DeleteRoleResponse{Success: true}, nil
}

func (h *AuthHandler) ListRoles(ctx context.Context, _ *authv1.ListRolesRequest) (*authv1.ListRolesResponse, error) {
	roles, err := h.q.ListRoles(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list roles")
	}
	out := make([]*authv1.Role, 0, len(roles))
	for _, r := range roles {
		perms, err := h.q.ListRolePermissionNames(ctx, r.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to list role permissions")
		}
		out = append(out, &authv1.Role{Id: r.ID, Name: r.Name, Description: r.Description, Permissions: perms})
	}
	return &authv1.ListRolesResponse{Roles: out}, nil
}

func (h *AuthHandler) AssignRole(ctx context.Context, req *authv1.AssignRoleRequest) (*authv1.AssignRoleResponse, error) {
	if err := requirePerm(ctx, "role:assign"); err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	if _, err := h.q.GetRoleByName(ctx, req.GetRoleName()); err != nil {
		return nil, status.Error(codes.NotFound, "role not found")
	}
	if err := h.q.AssignRoleToUser(ctx, db.AssignRoleToUserParams{UserID: userID, Name: req.GetRoleName()}); err != nil {
		return nil, status.Error(codes.Internal, "failed to assign role")
	}
	return &authv1.AssignRoleResponse{Success: true}, nil
}

func (h *AuthHandler) RevokeRole(ctx context.Context, req *authv1.RevokeRoleRequest) (*authv1.RevokeRoleResponse, error) {
	if err := requirePerm(ctx, "role:assign"); err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	if _, err := h.q.GetRoleByName(ctx, req.GetRoleName()); err != nil {
		return nil, status.Error(codes.NotFound, "role not found")
	}
	if err := h.q.RevokeRoleFromUser(ctx, db.RevokeRoleFromUserParams{UserID: userID, Name: req.GetRoleName()}); err != nil {
		return nil, status.Error(codes.Internal, "failed to revoke role")
	}
	return &authv1.RevokeRoleResponse{Success: true}, nil
}

func (h *AuthHandler) ListPermissions(ctx context.Context, _ *authv1.ListPermissionsRequest) (*authv1.ListPermissionsResponse, error) {
	perms, err := h.q.ListPermissions(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list permissions")
	}
	out := make([]*authv1.Permission, 0, len(perms))
	for _, p := range perms {
		out = append(out, &authv1.Permission{Id: p.ID, Name: p.Name, Description: p.Description})
	}
	return &authv1.ListPermissionsResponse{Permissions: out}, nil
}

func (h *AuthHandler) GrantPermission(ctx context.Context, req *authv1.GrantPermissionRequest) (*authv1.GrantPermissionResponse, error) {
	if err := requirePerm(ctx, "role:write"); err != nil {
		return nil, err
	}
	err := h.q.GrantPermissionToRole(ctx, db.GrantPermissionToRoleParams{Name: req.GetRoleName(), Name_2: req.GetPermissionName()})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to grant permission")
	}
	return &authv1.GrantPermissionResponse{Success: true}, nil
}

func (h *AuthHandler) RevokePermission(ctx context.Context, req *authv1.RevokePermissionRequest) (*authv1.RevokePermissionResponse, error) {
	if err := requirePerm(ctx, "role:write"); err != nil {
		return nil, err
	}
	err := h.q.RevokePermissionFromRole(ctx, db.RevokePermissionFromRoleParams{Name: req.GetRoleName(), Name_2: req.GetPermissionName()})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to revoke permission")
	}
	return &authv1.RevokePermissionResponse{Success: true}, nil
}

func isBuiltinRole(name string) bool {
	return name == "admin" || name == "user"
}

// ── helpers ─────────────────────────────────────────────────

func (h *AuthHandler) issueTokens(ctx context.Context, userID uuid.UUID, email string) (*authv1.TokenPair, error) {
	access, err := h.jwt.Issue(userID.String(), email)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to sign token")
	}
	refresh, err := newRefreshToken()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate refresh token")
	}
	expires := pgtype.Timestamptz{Time: time.Now().Add(h.refreshTTL), Valid: true}
	if _, err := h.q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID: userID, TokenHash: hashToken(refresh), ExpiresAt: expires,
	}); err != nil {
		return nil, status.Error(codes.Internal, "failed to persist refresh token")
	}
	return &authv1.TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(h.jwt.AccessTTL().Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func newRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
