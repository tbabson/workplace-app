package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"workplace/internal/asset"
	"workplace/internal/attendance"
	"workplace/internal/auth"
	"workplace/internal/chat"
	"workplace/internal/claim"
	"workplace/internal/department"
	"workplace/internal/leave"
	"workplace/internal/memo"
	"workplace/internal/milestone"
	"workplace/internal/notification"
	"workplace/internal/overtime"
	"workplace/internal/project"
	"workplace/internal/projectcomment"
	"workplace/internal/query"
	"workplace/internal/reminder"
	"workplace/internal/report"
	"workplace/internal/review"
	"workplace/internal/task"
	"workplace/internal/user"
	"workplace/pkg/cache"
	"workplace/pkg/config"
	"workplace/pkg/database"
	"workplace/pkg/middleware"
)

func main() {
	cfg := config.Load()

	db, err := database.NewPostgres(cfg.DatabaseURL, cfg.Location.String())
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer db.Close()

	rdb, err := database.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()

	appCache := cache.New(rdb)

	// ── Repositories ──────────────────────────────────────────────────────────
	userRepo := user.NewRepository(db)
	deptRepo := department.NewRepository(db)
	projectRepo := project.NewRepository(db)
	projectHistoryRepo := project.NewHistoryRepository(db)
	projectBudgetRepo := project.NewBudgetRepository(db)
	projectFileRepo := project.NewFileRepository(db)
	taskRepo := task.NewRepository(db)
	milestoneRepo := milestone.NewRepository(db)
	commentRepo := projectcomment.NewRepository(db)
	attendanceRepo := attendance.NewRepository(db)
	overtimeRepo := overtime.NewRepository(db)
	chatRepo := chat.NewRepository(db)
	memoRepo := memo.NewRepository(db)
	queryRepo := query.NewRepository(db)
	notifRepo := notification.NewRepository(db)
	leaveRepo := leave.NewRepository(db)
	reviewRepo := review.NewRepository(db)
	claimRepo := claim.NewRepository(db)
	assetRepo := asset.NewRepository(db)
	reportRepo := report.NewRepository(db)

	// ── Notification infrastructure ───────────────────────────────────────────
	mailer := notification.NewMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
	notifHub := notification.NewHub()
	notifSvc := notification.NewService(notifRepo, notifHub, mailer)

	// ── Services ──────────────────────────────────────────────────────────────
	userSvc := user.NewService(userRepo, appCache)
	authSvc := auth.NewService(userSvc, cfg.JWTSecret, cfg.JWTExpirationHrs, rdb)
	deptSvc := department.NewService(deptRepo, appCache)
	chatSvc := chat.NewService(chatRepo, userRepo)
	chatHub := chat.NewHub(chatRepo, userRepo, notifSvc)
	projectSvc := project.NewService(projectRepo, projectHistoryRepo, projectBudgetRepo, projectFileRepo, chatSvc, notifSvc, userRepo)
	taskSvc := task.NewService(taskRepo, notifSvc, chatSvc)
	milestoneSvc := milestone.NewService(milestoneRepo)
	commentSvc := projectcomment.NewService(commentRepo, notifSvc, projectRepo)
	attendanceSvc := attendance.NewService(attendanceRepo, notifSvc, userRepo, cfg.CompanyLat, cfg.CompanyLng, cfg.GeofenceRadius, cfg.DeviceLockEnabled)
	overtimeSvc := overtime.NewService(overtimeRepo, notifSvc, userRepo)
	memoSvc := memo.NewService(memoRepo, notifSvc, userRepo)
	querySvc := query.NewService(queryRepo, notifSvc, userRepo)
	leaveSvc := leave.NewService(leaveRepo, notifSvc)
	reviewSvc := review.NewService(reviewRepo)
	claimSvc := claim.NewService(claimRepo, notifSvc)
	assetSvc := asset.NewService(assetRepo, notifSvc)
	reportSvc := report.NewService(reportRepo, appCache)

	if err := userSvc.EnsureSuperAdmin(context.Background(), cfg.BootstrapAdminEmail, cfg.BootstrapAdminPassword); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}

	// ── Handlers ──────────────────────────────────────────────────────────────
	checkOrigin := middleware.OriginChecker(cfg.AllowedOrigins)
	h := handlers{
		auth:       auth.NewHandler(authSvc, !cfg.IsDevelopment()),
		user:       user.NewHandler(userSvc),
		dept:       department.NewHandler(deptSvc),
		project:    project.NewHandler(projectSvc),
		task:       task.NewHandler(taskSvc),
		milestone:  milestone.NewHandler(milestoneSvc),
		comment:    projectcomment.NewHandler(commentSvc),
		attendance: attendance.NewHandler(attendanceSvc),
		overtime:   overtime.NewHandler(overtimeSvc),
		chat:       chat.NewHandler(chatSvc, chatHub, checkOrigin),
		memo:       memo.NewHandler(memoSvc),
		query:      query.NewHandler(querySvc),
		notif:      notification.NewHandler(notifSvc, notifHub, checkOrigin),
		leave:      leave.NewHandler(leaveSvc),
		review:     review.NewHandler(reviewSvc),
		claim:      claim.NewHandler(claimSvc),
		asset:      asset.NewHandler(assetSvc),
		report:     report.NewHandler(reportSvc),
	}

	// ── Start hubs ────────────────────────────────────────────────────────────
	go chatHub.Run()
	go notifHub.Run()
	go reminder.New(projectRepo, taskRepo, attendanceRepo, notifSvc, rdb, cfg.Location).Run()

	// ── Server ────────────────────────────────────────────────────────────────
	authMw := middleware.Auth(cfg.JWTSecret, rdb, userSvc.Identity)
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           newRouter(h, authMw, cfg.AllowedOrigins),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		// WriteTimeout is deliberately unset: it would cut long-lived WebSocket
		// connections. Handlers are bounded by ReadTimeout and DB timeouts.
		IdleTimeout: 120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Server starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down…")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
