package main

import (
	"fmt"
	"log"
	"net/http"

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

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.Load()

	db, err := database.NewPostgres(cfg.DatabaseURL)
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
	userRepo           := user.NewRepository(db)
	deptRepo           := department.NewRepository(db)
	projectRepo        := project.NewRepository(db)
	projectHistoryRepo := project.NewHistoryRepository(db)
	projectBudgetRepo  := project.NewBudgetRepository(db)
	projectFileRepo    := project.NewFileRepository(db)
	taskRepo           := task.NewRepository(db)
	milestoneRepo      := milestone.NewRepository(db)
	commentRepo        := projectcomment.NewRepository(db)
	attendanceRepo     := attendance.NewRepository(db)
	overtimeRepo       := overtime.NewRepository(db)
	chatRepo           := chat.NewRepository(db)
	memoRepo           := memo.NewRepository(db)
	queryRepo          := query.NewRepository(db)
	notifRepo          := notification.NewRepository(db)
	leaveRepo          := leave.NewRepository(db)
	reviewRepo         := review.NewRepository(db)
	claimRepo          := claim.NewRepository(db)
	assetRepo          := asset.NewRepository(db)
	reportRepo         := report.NewRepository(db)

	// ── Notification infrastructure ───────────────────────────────────────────
	mailer   := notification.NewMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
	notifHub := notification.NewHub()
	notifSvc := notification.NewService(notifRepo, notifHub, mailer)

	// ── Services ──────────────────────────────────────────────────────────────
	userSvc       := user.NewService(userRepo, appCache)
	authSvc       := auth.NewService(userSvc, cfg.JWTSecret, cfg.JWTExpirationHrs, rdb)
	deptSvc       := department.NewService(deptRepo, appCache)
	chatSvc       := chat.NewService(chatRepo, userRepo)
	chatHub       := chat.NewHub(chatRepo, userRepo, notifSvc)
	projectSvc    := project.NewService(projectRepo, projectHistoryRepo, projectBudgetRepo, projectFileRepo, chatSvc, notifSvc, userRepo)
	taskSvc       := task.NewService(taskRepo, notifSvc, chatSvc)
	milestoneSvc  := milestone.NewService(milestoneRepo)
	commentSvc    := projectcomment.NewService(commentRepo, notifSvc, projectRepo)
	attendanceSvc := attendance.NewService(attendanceRepo, notifSvc, userRepo, cfg.CompanyLat, cfg.CompanyLng, cfg.GeofenceRadius, cfg.DeviceLockEnabled)
	overtimeSvc   := overtime.NewService(overtimeRepo, notifSvc, userRepo)
	memoSvc       := memo.NewService(memoRepo, notifSvc, userRepo)
	querySvc      := query.NewService(queryRepo, notifSvc, userRepo)
	leaveSvc      := leave.NewService(leaveRepo, notifSvc)
	reviewSvc     := review.NewService(reviewRepo)
	claimSvc      := claim.NewService(claimRepo, notifSvc)
	assetSvc      := asset.NewService(assetRepo, notifSvc)
	reportSvc     := report.NewService(reportRepo, appCache)

	// ── Handlers ──────────────────────────────────────────────────────────────
	authHandler       := auth.NewHandler(authSvc)
	userHandler       := user.NewHandler(userSvc)
	deptHandler       := department.NewHandler(deptSvc)
	projectHandler    := project.NewHandler(projectSvc)
	taskHandler       := task.NewHandler(taskSvc)
	milestoneHandler  := milestone.NewHandler(milestoneSvc)
	commentHandler    := projectcomment.NewHandler(commentSvc)
	attendanceHandler := attendance.NewHandler(attendanceSvc)
	overtimeHandler   := overtime.NewHandler(overtimeSvc)
	chatHandler       := chat.NewHandler(chatSvc, chatHub)
	memoHandler       := memo.NewHandler(memoSvc)
	queryHandler      := query.NewHandler(querySvc)
	notifHandler      := notification.NewHandler(notifSvc, notifHub)
	leaveHandler      := leave.NewHandler(leaveSvc)
	reviewHandler     := review.NewHandler(reviewSvc)
	claimHandler      := claim.NewHandler(claimSvc)
	assetHandler      := asset.NewHandler(assetSvc)
	reportHandler     := report.NewHandler(reportSvc)

	// ── Start hubs ────────────────────────────────────────────────────────────
	go chatHub.Run()
	go notifHub.Run()
	go reminder.New(projectRepo, taskRepo, attendanceRepo, notifSvc).Run()

	// ── Router ────────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(middleware.CORS)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	authMw := middleware.Auth(cfg.JWTSecret, rdb)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			authHandler.RegisterRoutes(r)
		})

		r.Group(func(r chi.Router) {
			r.Use(authMw)

			r.Route("/users",         userHandler.RegisterRoutes)
			r.Route("/departments",   deptHandler.RegisterRoutes)
			r.Route("/projects",      projectHandler.RegisterRoutes)
			taskHandler.RegisterRoutes(r)
			milestoneHandler.RegisterRoutes(r)
			commentHandler.RegisterRoutes(r)
			r.Route("/attendance",    attendanceHandler.RegisterRoutes)
			r.Route("/overtime",      overtimeHandler.RegisterRoutes)
			r.Route("/chat",          chatHandler.RegisterRoutes)
			r.Route("/memos",         memoHandler.RegisterRoutes)
			r.Route("/queries",       queryHandler.RegisterRoutes)
			r.Route("/notifications", notifHandler.RegisterRoutes)
			r.Route("/leaves",        leaveHandler.RegisterRoutes)
			r.Route("/reviews",       reviewHandler.RegisterRoutes)
			r.Route("/claims",        claimHandler.RegisterRoutes)
			r.Route("/assets",        assetHandler.RegisterRoutes)
			r.Route("/reports",       reportHandler.RegisterRoutes)
		})
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
