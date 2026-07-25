package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"firebase.google.com/go/auth"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrAdminUserNotFound means an account does not exist.
	ErrAdminUserNotFound = errors.New("admin user not found")
	// ErrAdminSelfDelete prevents an administrator from deleting their own account.
	ErrAdminSelfDelete = errors.New("administrators cannot delete their own account")
)

// AdminUserListParams controls account pagination and search.
type AdminUserListParams struct {
	Search string
	Limit  int
	Skip   int
}

// AdminUser is the safe operational view of an account.
type AdminUser struct {
	ID                               entities.UserID           `json:"id"`
	Email                            string                    `json:"email"`
	Timezone                         string                    `json:"timezone"`
	ActivePhoneID                    *uuid.UUID                `json:"active_phone_id"`
	SubscriptionName                 entities.SubscriptionName `json:"subscription_name"`
	SubscriptionLimit                uint                      `json:"subscription_limit"`
	NotificationMessageStatusEnabled bool                      `json:"notification_message_status_enabled"`
	NotificationWebhookEnabled       bool                      `json:"notification_webhook_enabled"`
	NotificationHeartbeatEnabled     bool                      `json:"notification_heartbeat_enabled"`
	NotificationNewsletterEnabled    bool                      `json:"notification_newsletter_enabled"`
	SentMessages                     int64                     `json:"sent_messages"`
	ReceivedMessages                 int64                     `json:"received_messages"`
	CurrentMessages                  int64                     `json:"current_messages"`
	PhoneCount                       int64                     `json:"phone_count"`
	ThreadCount                      int64                     `json:"thread_count"`
	WebhookCount                     int64                     `json:"webhook_count"`
	LastMessageAt                    *time.Time                `json:"last_message_at"`
	CreatedAt                        time.Time                 `json:"created_at"`
	UpdatedAt                        time.Time                 `json:"updated_at"`
}

// AdminUsers is a paginated account response.
type AdminUsers struct {
	Users []AdminUser `json:"users"`
	Total int64       `json:"total"`
}

// AdminCount is a named aggregate.
type AdminCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// AdminDailyUsage is one UTC day of activity.
type AdminDailyUsage struct {
	Date     string `json:"date"`
	Sent     int64  `json:"sent"`
	Received int64  `json:"received"`
	Total    int64  `json:"total"`
}

type adminDailyRow struct {
	Day      time.Time
	Sent     int64
	Received int64
}

// AdminTopUser is a ranked account usage result.
type AdminTopUser struct {
	ID       entities.UserID `json:"id"`
	Email    string          `json:"email"`
	Sent     int64           `json:"sent"`
	Received int64           `json:"received"`
	Total    int64           `json:"total"`
}

// AdminReportSummary contains report totals.
type AdminReportSummary struct {
	TotalMessages     int64   `json:"total_messages"`
	SentMessages      int64   `json:"sent_messages"`
	ReceivedMessages  int64   `json:"received_messages"`
	DeliveredMessages int64   `json:"delivered_messages"`
	FailedMessages    int64   `json:"failed_messages"`
	ExpiredMessages   int64   `json:"expired_messages"`
	ActiveUsers       int64   `json:"active_users"`
	NewUsers          int64   `json:"new_users"`
	DeliveryRate      float64 `json:"delivery_rate"`
}

// AdminReport contains report data for a date range.
type AdminReport struct {
	From            string             `json:"from"`
	To              string             `json:"to"`
	Summary         AdminReportSummary `json:"summary"`
	DailyUsage      []AdminDailyUsage  `json:"daily_usage"`
	StatusBreakdown []AdminCount       `json:"status_breakdown"`
	TopUsers        []AdminTopUser     `json:"top_users"`
}

// AdminOverview contains platform-wide operational totals.
type AdminOverview struct {
	TotalUsers         int64             `json:"total_users"`
	NewUsers30Days     int64             `json:"new_users_30_days"`
	ActiveUsers30Days  int64             `json:"active_users_30_days"`
	ConnectedPhones    int64             `json:"connected_phones"`
	TotalSent          int64             `json:"total_sent"`
	TotalReceived      int64             `json:"total_received"`
	TotalMessages      int64             `json:"total_messages"`
	Messages30Days     int64             `json:"messages_30_days"`
	Failed30Days       int64             `json:"failed_30_days"`
	DeliveryRate30Days float64           `json:"delivery_rate_30_days"`
	DailyUsage         []AdminDailyUsage `json:"daily_usage"`
	StatusBreakdown    []AdminCount      `json:"status_breakdown"`
	TopUsers           []AdminTopUser    `json:"top_users"`
}

// AdminUserDetail contains an account and its recent operational history.
type AdminUserDetail struct {
	User          AdminUser               `json:"user"`
	UsageHistory  []entities.BillingUsage `json:"usage_history"`
	MessageStatus []AdminCount            `json:"message_status"`
}

// AdminCreateUserParams contains fields for an administrator-created account.
type AdminCreateUserParams struct {
	Email            string
	Password         string
	Timezone         string
	SubscriptionName entities.SubscriptionName
}

// AdminUpdateUserParams contains editable account fields.
type AdminUpdateUserParams struct {
	Email                            string
	Password                         string
	Timezone                         string
	SubscriptionName                 entities.SubscriptionName
	NotificationMessageStatusEnabled bool
	NotificationWebhookEnabled       bool
	NotificationHeartbeatEnabled     bool
	NotificationNewsletterEnabled    bool
}

// AdminService powers the internal admin portal.
type AdminService struct {
	logger         telemetry.Logger
	tracer         telemetry.Tracer
	db             *gorm.DB
	userRepository repositories.UserRepository
	userService    *UserService
	authClient     *auth.Client
}

// NewAdminService creates an AdminService.
func NewAdminService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
	userRepository repositories.UserRepository,
	userService *UserService,
	authClient *auth.Client,
) *AdminService {
	return &AdminService{
		logger:         logger.WithService("services.AdminService"),
		tracer:         tracer,
		db:             db,
		userRepository: userRepository,
		userService:    userService,
		authClient:     authClient,
	}
}

// IsAdminSubscriptionName validates plan overrides accepted by the portal.
func IsAdminSubscriptionName(name entities.SubscriptionName) bool {
	switch name {
	case entities.SubscriptionNameFree,
		entities.SubscriptionNameProMonthly,
		entities.SubscriptionNameProYearly,
		entities.SubscriptionNameUltraMonthly,
		entities.SubscriptionNameUltraYearly,
		entities.SubscriptionNameProLifetime,
		entities.SubscriptionName20KMonthly,
		entities.SubscriptionName20KYearly,
		entities.SubscriptionName50KMonthly,
		entities.SubscriptionName100KMonthly,
		entities.SubscriptionName200KMonthly:
		return true
	default:
		return false
	}
}

func (service *AdminService) userQuery(ctx context.Context) *gorm.DB {
	now := time.Now().UTC()
	usage := service.db.WithContext(ctx).Table("billing_usages").
		Select("user_id, SUM(sent_messages) AS sent_messages, SUM(received_messages) AS received_messages").
		Group("user_id")
	currentUsage := service.db.WithContext(ctx).Table("billing_usages").
		Select("user_id, SUM(sent_messages + received_messages) AS current_messages").
		Where("start_timestamp <= ? AND end_timestamp >= ?", now, now).
		Group("user_id")
	phones := service.db.WithContext(ctx).Table("phones").
		Select("user_id, COUNT(*) AS phone_count").
		Group("user_id")
	threads := service.db.WithContext(ctx).Table("message_threads").
		Select("user_id, COUNT(*) AS thread_count").
		Group("user_id")
	webhooks := service.db.WithContext(ctx).Table("webhooks").
		Select("user_id, COUNT(*) AS webhook_count").
		Group("user_id")
	lastMessages := service.db.WithContext(ctx).Table("messages").
		Select("user_id, MAX(created_at) AS last_message_at").
		Group("user_id")

	return service.db.WithContext(ctx).Table("users AS u").
		Select(`u.id, u.email, u.timezone, u.active_phone_id, u.subscription_name,
			u.notification_message_status_enabled, u.notification_webhook_enabled,
			u.notification_heartbeat_enabled, u.notification_newsletter_enabled,
			u.created_at, u.updated_at,
			COALESCE(usage.sent_messages, 0) AS sent_messages,
			COALESCE(usage.received_messages, 0) AS received_messages,
			COALESCE(current_usage.current_messages, 0) AS current_messages,
			COALESCE(phones.phone_count, 0) AS phone_count,
			COALESCE(threads.thread_count, 0) AS thread_count,
			COALESCE(webhooks.webhook_count, 0) AS webhook_count,
			last_messages.last_message_at`).
		Joins("LEFT JOIN (?) AS usage ON usage.user_id = u.id", usage).
		Joins("LEFT JOIN (?) AS current_usage ON current_usage.user_id = u.id", currentUsage).
		Joins("LEFT JOIN (?) AS phones ON phones.user_id = u.id", phones).
		Joins("LEFT JOIN (?) AS threads ON threads.user_id = u.id", threads).
		Joins("LEFT JOIN (?) AS webhooks ON webhooks.user_id = u.id", webhooks).
		Joins("LEFT JOIN (?) AS last_messages ON last_messages.user_id = u.id", lastMessages)
}

func applyAdminUserLimits(users []AdminUser) {
	for index := range users {
		users[index].SubscriptionLimit = users[index].SubscriptionName.Limit()
	}
}

// Users lists registered accounts.
func (service *AdminService) Users(ctx context.Context, params AdminUserListParams) (*AdminUsers, error) {
	search := "%" + strings.ToLower(strings.TrimSpace(params.Search)) + "%"
	countQuery := service.db.WithContext(ctx).Model(&entities.User{})
	query := service.userQuery(ctx)
	if params.Search != "" {
		countQuery = countQuery.Where("LOWER(email) LIKE ? OR LOWER(CAST(id AS TEXT)) LIKE ?", search, search)
		query = query.Where("LOWER(u.email) LIKE ? OR LOWER(CAST(u.id AS TEXT)) LIKE ?", search, search)
	}

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	users := make([]AdminUser, 0)
	if err := query.Order("u.created_at DESC").Limit(params.Limit).Offset(params.Skip).Scan(&users).Error; err != nil {
		return nil, err
	}
	applyAdminUserLimits(users)
	return &AdminUsers{Users: users, Total: total}, nil
}

// User fetches one account with recent usage and message status details.
func (service *AdminService) User(ctx context.Context, userID entities.UserID) (*AdminUserDetail, error) {
	var users []AdminUser
	result := service.userQuery(ctx).Where("u.id = ?", userID).Limit(1).Scan(&users)
	if result.Error != nil {
		return nil, result.Error
	}
	if len(users) == 0 {
		return nil, ErrAdminUserNotFound
	}
	applyAdminUserLimits(users)

	history := make([]entities.BillingUsage, 0)
	if err := service.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("start_timestamp DESC").
		Limit(12).
		Find(&history).Error; err != nil {
		return nil, err
	}

	statuses := make([]AdminCount, 0)
	if err := service.db.WithContext(ctx).Table("messages").
		Select("CAST(status AS TEXT) AS name, COUNT(*) AS count").
		Where("user_id = ?", userID).
		Group("status").
		Order("count DESC").
		Scan(&statuses).Error; err != nil {
		return nil, err
	}

	return &AdminUserDetail{User: users[0], UsageHistory: history, MessageStatus: statuses}, nil
}

// CreateUser creates matching Firebase and application accounts.
func (service *AdminService) CreateUser(ctx context.Context, actor entities.AuthContext, params AdminCreateUserParams) (*AdminUserDetail, error) {
	email := strings.ToLower(strings.TrimSpace(params.Email))
	record, err := service.authClient.CreateUser(ctx, (&auth.UserToCreate{}).Email(email).Password(params.Password))
	if err != nil {
		return nil, err
	}

	user, _, err := service.userRepository.LoadOrStore(ctx, entities.AuthContext{
		ID:    entities.UserID(record.UID),
		Email: email,
	})
	if err != nil {
		_ = service.authClient.DeleteUser(ctx, record.UID)
		return nil, err
	}

	user.Timezone = params.Timezone
	user.SubscriptionName = params.SubscriptionName
	if err = service.userRepository.Update(ctx, user); err != nil {
		_ = service.userRepository.Delete(ctx, user)
		_ = service.authClient.DeleteUser(ctx, record.UID)
		return nil, err
	}

	service.audit(ctx, actor, "account.created", user.ID, fmt.Sprintf("Created %s on %s", user.Email, user.SubscriptionName))
	return service.User(ctx, user.ID)
}

// UpdateUser updates application and Firebase account data.
func (service *AdminService) UpdateUser(ctx context.Context, actor entities.AuthContext, userID entities.UserID, params AdminUpdateUserParams) (*AdminUserDetail, error) {
	user, err := service.userRepository.Load(ctx, userID)
	if err != nil {
		return nil, ErrAdminUserNotFound
	}
	original := *user

	user.Email = strings.ToLower(strings.TrimSpace(params.Email))
	user.Timezone = params.Timezone
	user.SubscriptionName = params.SubscriptionName
	user.NotificationMessageStatusEnabled = params.NotificationMessageStatusEnabled
	user.NotificationWebhookEnabled = params.NotificationWebhookEnabled
	user.NotificationHeartbeatEnabled = params.NotificationHeartbeatEnabled
	user.NotificationNewsletterEnabled = params.NotificationNewsletterEnabled
	if err = service.userRepository.Update(ctx, user); err != nil {
		return nil, err
	}

	update := (&auth.UserToUpdate{}).Email(user.Email)
	if params.Password != "" {
		update.Password(params.Password)
	}
	if _, err = service.authClient.UpdateUser(ctx, userID.String(), update); err != nil {
		_ = service.userRepository.Update(ctx, &original)
		return nil, err
	}

	service.audit(ctx, actor, "account.updated", user.ID, fmt.Sprintf("Updated %s; plan %s", user.Email, user.SubscriptionName))
	return service.User(ctx, user.ID)
}

// DeleteUser deletes an account through the existing cleanup flow.
func (service *AdminService) DeleteUser(ctx context.Context, source string, actor entities.AuthContext, userID entities.UserID) error {
	if actor.ID == userID {
		return ErrAdminSelfDelete
	}
	user, err := service.userRepository.Load(ctx, userID)
	if err != nil {
		return ErrAdminUserNotFound
	}
	if err = service.userService.Delete(ctx, source, userID); err != nil {
		return err
	}
	service.audit(ctx, actor, "account.deleted", userID, fmt.Sprintf("Deleted %s", user.Email))
	return nil
}

// RotateAPIKey rotates an account API key without returning the secret to the portal.
func (service *AdminService) RotateAPIKey(ctx context.Context, source string, actor entities.AuthContext, userID entities.UserID) (*AdminUserDetail, error) {
	if _, err := service.userService.RotateAPIKey(ctx, source, userID); err != nil {
		return nil, err
	}
	service.audit(ctx, actor, "api_key.rotated", userID, "Rotated primary API key")
	return service.User(ctx, userID)
}

// Overview returns app-wide operational totals.
func (service *AdminService) Overview(ctx context.Context) (*AdminOverview, error) {
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -29).Truncate(24 * time.Hour)
	to := now.AddDate(0, 0, 1).Truncate(24 * time.Hour)
	report, err := service.Report(ctx, from, to)
	if err != nil {
		return nil, err
	}

	overview := &AdminOverview{
		ActiveUsers30Days:  report.Summary.ActiveUsers,
		Messages30Days:     report.Summary.TotalMessages,
		Failed30Days:       report.Summary.FailedMessages + report.Summary.ExpiredMessages,
		DeliveryRate30Days: report.Summary.DeliveryRate,
		DailyUsage:         report.DailyUsage,
		StatusBreakdown:    report.StatusBreakdown,
		TopUsers:           report.TopUsers,
	}
	if err = service.db.WithContext(ctx).Model(&entities.User{}).Count(&overview.TotalUsers).Error; err != nil {
		return nil, err
	}
	if err = service.db.WithContext(ctx).Model(&entities.User{}).Where("created_at >= ?", from).Count(&overview.NewUsers30Days).Error; err != nil {
		return nil, err
	}
	if err = service.db.WithContext(ctx).Model(&entities.Phone{}).Count(&overview.ConnectedPhones).Error; err != nil {
		return nil, err
	}
	totals := struct {
		TotalSent     int64
		TotalReceived int64
	}{}
	if err = service.db.WithContext(ctx).Table("billing_usages").
		Select("COALESCE(SUM(sent_messages), 0) AS total_sent, COALESCE(SUM(received_messages), 0) AS total_received").
		Scan(&totals).Error; err != nil {
		return nil, err
	}
	overview.TotalSent = totals.TotalSent
	overview.TotalReceived = totals.TotalReceived
	overview.TotalMessages = overview.TotalSent + overview.TotalReceived
	return overview, nil
}

// Report returns activity for an inclusive UTC date range represented by [from, to).
func (service *AdminService) Report(ctx context.Context, from, to time.Time) (*AdminReport, error) {
	report := &AdminReport{
		From: from.Format(time.DateOnly),
		To:   to.Add(-time.Second).Format(time.DateOnly),
	}

	if err := service.db.WithContext(ctx).Table("messages").
		Select(`COUNT(*) AS total_messages,
			COALESCE(SUM(CASE WHEN type = ? THEN 1 ELSE 0 END), 0) AS sent_messages,
			COALESCE(SUM(CASE WHEN type <> ? THEN 1 ELSE 0 END), 0) AS received_messages,
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS delivered_messages,
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS failed_messages,
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS expired_messages,
			COUNT(DISTINCT user_id) AS active_users`,
			entities.MessageTypeMobileTerminated,
			entities.MessageTypeMobileTerminated,
			entities.MessageStatusDelivered,
			entities.MessageStatusFailed,
			entities.MessageStatusExpired).
		Where("created_at >= ? AND created_at < ?", from, to).
		Scan(&report.Summary).Error; err != nil {
		return nil, err
	}
	if err := service.db.WithContext(ctx).Model(&entities.User{}).
		Where("created_at >= ? AND created_at < ?", from, to).
		Count(&report.Summary.NewUsers).Error; err != nil {
		return nil, err
	}
	terminal := report.Summary.DeliveredMessages + report.Summary.FailedMessages + report.Summary.ExpiredMessages
	if terminal > 0 {
		report.Summary.DeliveryRate = float64(report.Summary.DeliveredMessages) * 100 / float64(terminal)
	}

	rows := make([]adminDailyRow, 0)
	if err := service.db.WithContext(ctx).Table("messages").
		Select(`DATE(created_at AT TIME ZONE 'UTC') AS day,
			COALESCE(SUM(CASE WHEN type = ? THEN 1 ELSE 0 END), 0) AS sent,
			COALESCE(SUM(CASE WHEN type <> ? THEN 1 ELSE 0 END), 0) AS received`,
			entities.MessageTypeMobileTerminated,
			entities.MessageTypeMobileTerminated).
		Where("created_at >= ? AND created_at < ?", from, to).
		Group("day").
		Order("day").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	report.DailyUsage = fillAdminDailyUsage(from, to, rows)

	report.StatusBreakdown = make([]AdminCount, 0)
	if err := service.db.WithContext(ctx).Table("messages").
		Select("CAST(status AS TEXT) AS name, COUNT(*) AS count").
		Where("created_at >= ? AND created_at < ?", from, to).
		Group("status").
		Order("count DESC").
		Scan(&report.StatusBreakdown).Error; err != nil {
		return nil, err
	}

	report.TopUsers = make([]AdminTopUser, 0)
	if err := service.db.WithContext(ctx).Table("messages AS m").
		Select(`u.id, u.email,
			COALESCE(SUM(CASE WHEN m.type = ? THEN 1 ELSE 0 END), 0) AS sent,
			COALESCE(SUM(CASE WHEN m.type <> ? THEN 1 ELSE 0 END), 0) AS received,
			COUNT(*) AS total`,
			entities.MessageTypeMobileTerminated,
			entities.MessageTypeMobileTerminated).
		Joins("JOIN users AS u ON u.id = m.user_id").
		Where("m.created_at >= ? AND m.created_at < ?", from, to).
		Group("u.id, u.email").
		Order("total DESC").
		Limit(10).
		Scan(&report.TopUsers).Error; err != nil {
		return nil, err
	}

	return report, nil
}

func fillAdminDailyUsage(from, to time.Time, rows []adminDailyRow) []AdminDailyUsage {
	byDay := make(map[string]AdminDailyUsage, len(rows))
	for _, row := range rows {
		key := row.Day.Format(time.DateOnly)
		byDay[key] = AdminDailyUsage{Date: key, Sent: row.Sent, Received: row.Received, Total: row.Sent + row.Received}
	}

	result := make([]AdminDailyUsage, 0, int(to.Sub(from).Hours()/24))
	for day := from; day.Before(to); day = day.AddDate(0, 0, 1) {
		key := day.Format(time.DateOnly)
		value, ok := byDay[key]
		if !ok {
			value = AdminDailyUsage{Date: key}
		}
		result = append(result, value)
	}
	return result
}

// AuditLogs returns the latest administrator actions.
func (service *AdminService) AuditLogs(ctx context.Context, limit int) ([]entities.AdminAuditLog, error) {
	logs := make([]entities.AdminAuditLog, 0)
	err := service.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

func (service *AdminService) audit(ctx context.Context, actor entities.AuthContext, action string, target entities.UserID, details string) {
	log := entities.AdminAuditLog{
		ID:           uuid.New(),
		AdminUserID:  actor.ID,
		AdminEmail:   actor.Email,
		Action:       action,
		TargetUserID: target,
		Details:      details,
		CreatedAt:    time.Now().UTC(),
	}
	if err := service.db.WithContext(ctx).Create(&log).Error; err != nil {
		service.logger.Error(fmt.Errorf("cannot write admin audit log: %w", err))
	}
}
