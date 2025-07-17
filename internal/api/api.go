package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"log"
	"micro-CRM/internal/database"
	"micro-CRM/internal/handlers"
	"micro-CRM/internal/logger"
	"micro-CRM/internal/middleware"
	"micro-CRM/internal/models"
	"micro-CRM/internal/oidc"
	_ "micro-CRM/internal/oidc"
	"micro-CRM/internal/utils"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Api struct {
	db          *sql.DB
	runningPort string
	jwtToken    string
	router      *mux.Router
	authRouter  *mux.Router
	dashRouter  *mux.Router
	adminRouter *mux.Router
	Params      models.EnvParams
	handlers.CRMHandlers
	database.DBManager
	log logger.Logger
}

func NewApi(p models.EnvParams) *Api {
	return &Api{
		Params: p,
	}
}
func (a *Api) SetupAuthRouter() {
	a.authRouter = a.router.PathPrefix("/api").Subrouter()
	a.authRouter.Use(middleware.AuthMiddleware)
}
func (a *Api) SetupDashRouter() {
	a.dashRouter = a.router.PathPrefix("/dash").Subrouter()
	a.dashRouter.Use(middleware.AuthMiddleware)
}
func (a *Api) SetupAdminRouter() {
	a.adminRouter = a.router.PathPrefix("/admin").Subrouter()
	a.dashRouter.Use(middleware.AuthMiddleware)
}
func (a *Api) SetupAllRoutes() {
	// Setup auth router
	a.SetupAuthRouter()

	// Setup Dashboard router
	a.SetupDashRouter()

	// Setup Admin router
	a.SetupAdminRouter()

	// Setup routes
	a.SetupAuthenticationRoutes()
	a.SetupCompanyRoutes()
	a.SetupContactRoutes()
	a.SetupFileRoutes()
	a.SetupTaskRoutes()
	a.SetupAdminRoutes()
	a.SetupInteractionRoutes()
	a.SetupDashboardRoutes()
	a.SetupProfileRoutes()
}
func (a *Api) SetupAuthenticationRoutes() {
	a.router.HandleFunc("/register", a.CRMHandlers.RegisterUser).Methods("POST")
	a.router.HandleFunc("/login", a.CRMHandlers.LoginUser).Methods("POST")
	a.router.HandleFunc("/login/oidc", a.CRMHandlers.OIDCLoginHandler).Methods("GET")
	a.router.HandleFunc("/login/oidc/callback", a.CRMHandlers.OIDCCallbackHandler).Methods("GET")
	// a.authRouter.HandleFunc("/logout/oidc", a.CRMHandlers.OIDCLogoutHandler).Methods("GET")
}
func (a *Api) SetupOIDC() error {
	return oidc.InitOIDC(context.Background())
}
func (a *Api) SetupCompanyRoutes() {
	a.authRouter.HandleFunc("/companies", a.CRMHandlers.CreateCompany).Methods("POST")
	a.authRouter.HandleFunc("/companies", a.CRMHandlers.ListCompanies).Methods("GET")
	a.authRouter.HandleFunc("/companies/{id}", a.CRMHandlers.GetCompany).Methods("GET")
	a.authRouter.HandleFunc("/companies/{id}", a.CRMHandlers.UpdateCompany).Methods("PUT")
	a.authRouter.HandleFunc("/companies/{id}", a.CRMHandlers.DeleteCompany).Methods("DELETE")
}
func (a *Api) SetupContactRoutes() {
	a.authRouter.HandleFunc("/contacts", a.CRMHandlers.CreateContact).Methods("POST")
	a.authRouter.HandleFunc("/contacts", a.CRMHandlers.ListContacts).Methods("GET")
	a.authRouter.HandleFunc("/contacts/{id}", a.CRMHandlers.GetContact).Methods("GET")
	a.authRouter.HandleFunc("/contacts/{id}", a.CRMHandlers.UpdateContact).Methods("PUT")
	a.authRouter.HandleFunc("/contacts/{id}", a.CRMHandlers.DeleteContact).Methods("DELETE")
}
func (a *Api) SetupFileRoutes() {
	// a.authRouter.HandleFunc("/files", a.CRMHandlers.CreateFile).Methods("POST") # Will reuse this later
	a.authRouter.HandleFunc("/files", a.CRMHandlers.ListFiles).Methods("GET")
	a.authRouter.HandleFunc("/files/upload", a.CRMHandlers.UploadFileHandler).Methods("POST")
	a.authRouter.HandleFunc("/files/{id}", a.CRMHandlers.GetFile).Methods("GET")
	a.authRouter.HandleFunc("/files/{id}", a.CRMHandlers.UpdateFile).Methods("PUT")
	a.authRouter.HandleFunc("/files/{id}", a.CRMHandlers.DeleteFile).Methods("DELETE")
	a.authRouter.HandleFunc("/files/{id}/download", a.CRMHandlers.DownloadFileHandler).Methods("GET")
	a.authRouter.HandleFunc("/files/{id}/view", a.CRMHandlers.ViewFileHandler).Methods("GET")

	// TODO: restructure with new admin router in mind
	a.adminRouter.HandleFunc("/files/cleanup", a.CRMHandlers.CleanupOrphanedFiles).Methods("DELETE")
}
func (a *Api) SetupProfileRoutes() {
	a.authRouter.HandleFunc("/profile", a.CRMHandlers.GetUserInfo).Methods("GET")
	a.authRouter.HandleFunc("/profile", a.CRMHandlers.UpdateUserInfo).Methods("PUT")
	a.authRouter.HandleFunc("/profile", a.CRMHandlers.DeleteUser).Methods("DELETE")
	a.authRouter.HandleFunc("/profile/stats", a.GetProfileStats).Methods("GET")
}
func (a *Api) SetupTaskRoutes() {
	a.authRouter.HandleFunc("/tasks", a.CRMHandlers.CreateTask).Methods("POST")
	a.authRouter.HandleFunc("/tasks", a.CRMHandlers.ListTasks).Methods("GET")
	a.authRouter.HandleFunc("/tasks/{id}", a.CRMHandlers.GetTask).Methods("GET")
	a.authRouter.HandleFunc("/tasks/{id}", a.CRMHandlers.UpdateTask).Methods("PUT")
	a.authRouter.HandleFunc("/tasks/{id}", a.CRMHandlers.DeleteTask).Methods("DELETE")
}
func (a *Api) SetupInteractionRoutes() {
	a.authRouter.HandleFunc("/interactions", a.CRMHandlers.CreateInteraction).Methods("POST")
	a.authRouter.HandleFunc("/interactions", a.CRMHandlers.ListInteractions).Methods("GET")
	a.authRouter.HandleFunc("/interactions/{id}", a.CRMHandlers.GetInteraction).Methods("GET")
	a.authRouter.HandleFunc("/interactions/{id}", a.CRMHandlers.UpdateInteraction).Methods("PUT")
	a.authRouter.HandleFunc("/interactions/{id}", a.CRMHandlers.DeleteInteraction).Methods("DELETE")
}
func (a *Api) SetupDashboardRoutes() {
	a.dashRouter.Use(middleware.AuthMiddleware)
	a.dashRouter.HandleFunc("/stats", a.CRMHandlers.GetDashboardStats).Methods("GET")
	a.dashRouter.HandleFunc("/pipeline", a.CRMHandlers.GetPipelineData).Methods("GET")
	a.dashRouter.HandleFunc("/interactions", a.CRMHandlers.GetInteractionTrends).Methods("GET")
	a.dashRouter.HandleFunc("/recent-interactions", a.CRMHandlers.GetRecentInteractions).Methods("GET")
	a.dashRouter.HandleFunc("/suggested-contacts", a.CRMHandlers.GetSuggestedContacts).Methods("GET")
}
func (a *Api) SetupLogger() {
	a.log = logger.NewConsoleLogger(os.Stderr, "[CRM-API]", 0, logger.LogLevelInfo)
	a.CRMHandlers.Log = a.log
	a.DBManager.Log = a.log
	a.log.Info("Custom Logger initialized")
}
func (a *Api) SetupAdminRoutes() {
	a.adminRouter.HandleFunc("/health/API", a.CRMHandlers.Hello).Methods("GET")
	a.adminRouter.HandleFunc("/health/DB", a.CRMHandlers.DBPing).Methods("GET")
}
func (a *Api) SetupDatabases() {
	a.log.Info("Setting up API databases")
	manager := database.NewDBManager(a.Params.DbPath)

	if err := manager.Connect(); err != nil {
		a.log.Fatal("Cannot connect to database")
	}

	if err := manager.ApplyMigrations(); err != nil {
		a.log.Fatal("Cannot Apply Migrations : ", err)
	}

	a.log.Info("Setting up token storage")
	tokenStore, err := manager.InitTokenStore()
	if err != nil {
		a.log.Fatal("Cannot initialize token storage : ", err)
	}

	// ✅ This must be done unconditionally
	a.db = manager.DB
	a.CRMHandlers.DB = manager.DB
	a.CRMHandlers.TokenStore = tokenStore

	a.log.Info("DB setup complete")
}
func (a *Api) Start() {
	var (
		startErr error
	)
	fmt.Println(models.StartupText)
	// Wait a Second
	time.Sleep(50 * time.Millisecond)

	// Setting Up logger
	a.SetupLogger()
	// Token setting
	a.log.Info("Setting JWT token")
	utils.SetJWTSecret(a.Params.JWTToken)

	// Initialize OIDC if variables present
	if !utils.IsOidcMissing(utils.GetAllOidcParams()) {
		a.log.Info("Setting Up OIDC")
		startErr = a.SetupOIDC()
		if startErr != nil {
			a.log.Warn("Cannot start OIDC functionality : ", startErr)
		}
	}

	// Database Setup
	a.SetupDatabases()

	// Router initialization
	a.router = mux.NewRouter()

	// Setup routes
	a.log.Info("Setting up routes")
	a.SetupAllRoutes()
	// Kill channel
	var killSignal = make(chan os.Signal)
	signal.Notify(killSignal, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGKILL)
	handler := cors.AllowAll().Handler(a.router)
	server := &http.Server{
		Addr:    ":" + a.Params.ApiPort,
		Handler: handler,
	}

	go func() {
		startSting := "Starting API at endpoint: " + a.Params.ApiPort
		a.log.Info(startSting)
		if err := server.ListenAndServeTLS(a.Params.CertFilePath, a.Params.KeyFilePath); err != nil && !errors.Is(http.ErrServerClosed, err) {
			log.Fatalf("Cannot start API: %v", err)
		}
	}()

	<-killSignal // Block until a signal is received
	log.Println("Received shutdown signal. Initiating graceful shutdown...")

	// Context creation and graceful shutdown of server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 5 seconds to shut down
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	a.Stop()
}
func (a *Api) Stop() {
	a.log.Info("Graceful shutdown of services")
	err := a.db.Close()
	err = a.TokenStore.DB.Close()
	if err != nil {
		a.log.Warn("Cannot close token storage : ", err)
	}
	if err != nil {
		a.log.Warn("Couldn't close database connection:", err)
	}
	a.log.Info("API shutdown gracefully")
}
