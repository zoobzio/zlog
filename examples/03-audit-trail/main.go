package main

import (
	"os"

	"github.com/zoobzio/zlog"
)

func main() {
	// Standard logs go to stderr
	zlog.EnableStandardLogging(os.Stderr)

	// Audit logs go to a separate file
	auditFile, err := os.Create("audit.log")
	if err != nil {
		zlog.Fatal("Failed to create audit file", zlog.Err(err))
	}
	defer auditFile.Close()

	// Enable audit logging (handles both AUDIT and SECURITY signals)
	zlog.EnableAuditLogging(auditFile)

	// Start the application
	zlog.Info("Service started")

	// Simulate user service initialization
	userService := &UserService{}
	userService.Init()

	// Simulate some user actions
	userService.Login("alice", "192.168.1.100", true)
	userService.CheckPermission("alice", "admin_panel", "view", false)
	userService.Login("bob", "10.0.0.50", false)
	userService.UpdateProfile("alice", map[string]string{"email": "alice@example.com"})
	userService.DeleteAccount("charlie", "user_requested")

	zlog.Info("Service shutdown")
}

type UserService struct{}

func (s *UserService) Init() {
	zlog.Info("User service ready")
}

func (s *UserService) Login(userID, ip string, success bool) {
	if success {
		// Successful login is an audit event
		zlog.Emit(zlog.AUDIT, "User login",
			zlog.String("user_id", userID),
			zlog.String("ip", ip),
			zlog.Bool("success", success),
		)
	} else {
		// Failed login is a security event
		zlog.Emit(zlog.SECURITY, "Failed login attempt",
			zlog.String("user_id", userID),
			zlog.String("ip", ip),
			zlog.String("reason", "invalid_password"),
		)
	}
}

func (s *UserService) CheckPermission(userID, resource, action string, granted bool) {
	// All permission checks are audit events
	zlog.Emit(zlog.AUDIT, "Permission check",
		zlog.String("user_id", userID),
		zlog.String("resource", resource),
		zlog.String("action", action),
		zlog.Bool("granted", granted),
	)
}

func (s *UserService) UpdateProfile(userID string, changes map[string]string) {
	// Profile updates are audit events
	fields := []zlog.Field{
		zlog.String("user_id", userID),
		zlog.String("action", "profile_update"),
	}

	// Add what changed
	for key, value := range changes {
		fields = append(fields, zlog.String("changed_"+key, value))
	}

	zlog.Emit(zlog.AUDIT, "User profile updated", fields...)
}

func (s *UserService) DeleteAccount(userID, reason string) {
	// Account deletion is both audit and security event
	fields := []zlog.Field{
		zlog.String("user_id", userID),
		zlog.String("action", "account_deletion"),
		zlog.String("reason", reason),
	}

	// Log to audit trail
	zlog.Emit(zlog.AUDIT, "Account deleted", fields...)

	// Also log as security event for monitoring
	zlog.Emit(zlog.SECURITY, "Account deletion", fields...)
}

func init() {
	// Clean up any existing audit.log from previous runs
	os.Remove("audit.log")
}
