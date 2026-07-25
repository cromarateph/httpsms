package handlers

import (
	"errors"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"firebase.google.com/go/auth"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/gofiber/fiber/v3"
)

// AdminHandler handles internal administrator requests.
type AdminHandler struct {
	handler
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.AdminService
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(logger telemetry.Logger, tracer telemetry.Tracer, service *services.AdminService) *AdminHandler {
	return &AdminHandler{
		logger:  logger.WithService("handlers.AdminHandler"),
		tracer:  tracer,
		service: service,
	}
}

// RegisterRoutes registers the admin portal API.
func (h *AdminHandler) RegisterRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	h.register(router, fiber.MethodGet, "/v1/admin/access", middlewares, h.Access)
	h.register(router, fiber.MethodGet, "/v1/admin/overview", middlewares, h.Overview)
	h.register(router, fiber.MethodGet, "/v1/admin/users", middlewares, h.Users)
	h.register(router, fiber.MethodPost, "/v1/admin/users", middlewares, h.CreateUser)
	h.register(router, fiber.MethodGet, "/v1/admin/users/:userID", middlewares, h.User)
	h.register(router, fiber.MethodPut, "/v1/admin/users/:userID", middlewares, h.UpdateUser)
	h.register(router, fiber.MethodDelete, "/v1/admin/users/:userID", middlewares, h.DeleteUser)
	h.register(router, fiber.MethodPost, "/v1/admin/users/:userID/rotate-api-key", middlewares, h.RotateAPIKey)
	h.register(router, fiber.MethodGet, "/v1/admin/reports", middlewares, h.Report)
	h.register(router, fiber.MethodGet, "/v1/admin/audit-logs", middlewares, h.AuditLogs)
}

// Access confirms administrator access.
func (h *AdminHandler) Access(c fiber.Ctx) error {
	user := h.userFromContext(c)
	return h.responseOK(c, "administrator access confirmed", fiber.Map{
		"id":    user.ID,
		"email": user.Email,
	})
}

// Overview returns app-wide statistics.
func (h *AdminHandler) Overview(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	overview, err := h.service.Overview(ctx)
	if err != nil {
		h.tracer.CtxLogger(h.logger, span).Error(stacktrace.Propagate(err, "cannot fetch admin overview"))
		return h.responseInternalServerError(c)
	}
	return h.responseOK(c, "admin overview fetched", overview)
}

// Users lists registered accounts.
func (h *AdminHandler) Users(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	limit := queryInt(c, "limit", 25)
	if limit < 1 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	skip := queryInt(c, "skip", 0)
	if skip < 0 {
		skip = 0
	}

	users, err := h.service.Users(ctx, services.AdminUserListParams{
		Search: c.Query("search"),
		Limit:  limit,
		Skip:   skip,
	})
	if err != nil {
		h.tracer.CtxLogger(h.logger, span).Error(stacktrace.Propagate(err, "cannot list admin users"))
		return h.responseInternalServerError(c)
	}
	return h.responseOK(c, "registered accounts fetched", users)
}

// User returns one registered account.
func (h *AdminHandler) User(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	user, err := h.service.User(ctx, entities.UserID(c.Params("userID")))
	if err != nil {
		return h.handleUserError(c, err)
	}
	return h.responseOK(c, "account fetched", user)
}

type adminCreateUserRequest struct {
	Email            string                    `json:"email"`
	Password         string                    `json:"password"`
	Timezone         string                    `json:"timezone"`
	SubscriptionName entities.SubscriptionName `json:"subscription_name"`
}

// CreateUser creates a Firebase and application account.
func (h *AdminHandler) CreateUser(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	var request adminCreateUserRequest
	if err := c.Bind().Body(&request); err != nil {
		return h.responseBadRequest(c, err)
	}
	if validation := validateAdminAccount(request.Email, request.Password, request.Timezone, request.SubscriptionName, true); len(validation) > 0 {
		return h.responseUnprocessableEntity(c, validation, "account details are invalid")
	}

	user, err := h.service.CreateUser(ctx, h.userFromContext(c), services.AdminCreateUserParams{
		Email:            request.Email,
		Password:         request.Password,
		Timezone:         request.Timezone,
		SubscriptionName: request.SubscriptionName,
	})
	if err != nil {
		if auth.IsEmailAlreadyExists(err) {
			return h.responseUnprocessableEntity(c, url.Values{"email": {"An account with this email already exists."}}, "account details are invalid")
		}
		h.tracer.CtxLogger(h.logger, span).Error(stacktrace.Propagate(err, "cannot create admin user"))
		return h.responseInternalServerError(c)
	}
	return h.responseCreated(c, "account created", user)
}

type adminUpdateUserRequest struct {
	Email                            string                    `json:"email"`
	Password                         string                    `json:"password"`
	Timezone                         string                    `json:"timezone"`
	SubscriptionName                 entities.SubscriptionName `json:"subscription_name"`
	NotificationMessageStatusEnabled bool                      `json:"notification_message_status_enabled"`
	NotificationWebhookEnabled       bool                      `json:"notification_webhook_enabled"`
	NotificationHeartbeatEnabled     bool                      `json:"notification_heartbeat_enabled"`
	NotificationNewsletterEnabled    bool                      `json:"notification_newsletter_enabled"`
}

// UpdateUser updates account settings and Firebase credentials.
func (h *AdminHandler) UpdateUser(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	var request adminUpdateUserRequest
	if err := c.Bind().Body(&request); err != nil {
		return h.responseBadRequest(c, err)
	}
	if validation := validateAdminAccount(request.Email, request.Password, request.Timezone, request.SubscriptionName, false); len(validation) > 0 {
		return h.responseUnprocessableEntity(c, validation, "account details are invalid")
	}

	user, err := h.service.UpdateUser(ctx, h.userFromContext(c), entities.UserID(c.Params("userID")), services.AdminUpdateUserParams{
		Email:                            request.Email,
		Password:                         request.Password,
		Timezone:                         request.Timezone,
		SubscriptionName:                 request.SubscriptionName,
		NotificationMessageStatusEnabled: request.NotificationMessageStatusEnabled,
		NotificationWebhookEnabled:       request.NotificationWebhookEnabled,
		NotificationHeartbeatEnabled:     request.NotificationHeartbeatEnabled,
		NotificationNewsletterEnabled:    request.NotificationNewsletterEnabled,
	})
	if err != nil {
		if auth.IsEmailAlreadyExists(err) {
			return h.responseUnprocessableEntity(c, url.Values{"email": {"An account with this email already exists."}}, "account details are invalid")
		}
		return h.handleUserError(c, err)
	}
	return h.responseOK(c, "account updated", user)
}

// DeleteUser removes an account through the existing cleanup process.
func (h *AdminHandler) DeleteUser(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	err := h.service.DeleteUser(ctx, c.OriginalURL(), h.userFromContext(c), entities.UserID(c.Params("userID")))
	if err != nil {
		return h.handleUserError(c, err)
	}
	return h.responseNoContent(c, "account deleted")
}

// RotateAPIKey rotates a user's primary API key.
func (h *AdminHandler) RotateAPIKey(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	user, err := h.service.RotateAPIKey(ctx, c.OriginalURL(), h.userFromContext(c), entities.UserID(c.Params("userID")))
	if err != nil {
		return h.handleUserError(c, err)
	}
	return h.responseOK(c, "API key rotated", user)
}

// Report returns filtered platform activity.
func (h *AdminHandler) Report(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	from, to, validation := adminReportPeriod(c.Query("from"), c.Query("to"), time.Now().UTC())
	if len(validation) > 0 {
		return h.responseUnprocessableEntity(c, validation, "report dates are invalid")
	}
	report, err := h.service.Report(ctx, from, to)
	if err != nil {
		h.tracer.CtxLogger(h.logger, span).Error(stacktrace.Propagate(err, "cannot create admin report"))
		return h.responseInternalServerError(c)
	}
	return h.responseOK(c, "admin report fetched", report)
}

// AuditLogs returns recent administrator activity.
func (h *AdminHandler) AuditLogs(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	limit := queryInt(c, "limit", 100)
	if limit < 1 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	logs, err := h.service.AuditLogs(ctx, limit)
	if err != nil {
		h.tracer.CtxLogger(h.logger, span).Error(stacktrace.Propagate(err, "cannot fetch admin audit logs"))
		return h.responseInternalServerError(c)
	}
	return h.responseOK(c, "administrator activity fetched", logs)
}

func (h *AdminHandler) handleUserError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, services.ErrAdminUserNotFound):
		return h.responseNotFound(c, "Account not found.")
	case errors.Is(err, services.ErrAdminSelfDelete):
		return h.responseUnprocessableEntity(c, url.Values{"account": {err.Error()}}, "account cannot be deleted")
	default:
		return h.responseInternalServerError(c)
	}
}

func queryInt(c fiber.Ctx, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return fallback
	}
	return value
}

func validateAdminAccount(email, password, timezone string, subscription entities.SubscriptionName, passwordRequired bool) url.Values {
	validation := url.Values{}
	email = strings.TrimSpace(email)
	address, err := mail.ParseAddress(email)
	if err != nil || !strings.EqualFold(address.Address, email) {
		validation.Add("email", "Enter a valid email address.")
	}
	if (passwordRequired || password != "") && len(password) < 8 {
		validation.Add("password", "Password must contain at least 8 characters.")
	}
	if _, err = time.LoadLocation(timezone); err != nil {
		validation.Add("timezone", "Enter a valid IANA timezone.")
	}
	if !services.IsAdminSubscriptionName(subscription) {
		validation.Add("subscription_name", "Select a valid account limit.")
	}
	return validation
}

func adminReportPeriod(fromValue, toValue string, now time.Time) (time.Time, time.Time, url.Values) {
	validation := url.Values{}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	from := today.AddDate(0, 0, -29)
	to := today
	var err error
	if fromValue != "" {
		from, err = time.Parse(time.DateOnly, fromValue)
		if err != nil {
			validation.Add("from", "Use YYYY-MM-DD.")
		}
	}
	if toValue != "" {
		to, err = time.Parse(time.DateOnly, toValue)
		if err != nil {
			validation.Add("to", "Use YYYY-MM-DD.")
		}
	}
	if len(validation) == 0 {
		if to.Before(from) {
			validation.Add("to", "End date must be on or after the start date.")
		} else if to.Sub(from) > 365*24*time.Hour {
			validation.Add("to", "Reports are limited to 366 days.")
		}
	}
	return from, to.AddDate(0, 0, 1), validation
}
