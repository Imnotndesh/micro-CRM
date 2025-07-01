package api

import (
	"context"
	"database/sql"
	"errors"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"log"
	"micro-CRM/internal/database"
	"micro-CRM/internal/handlers"
	"micro-CRM/internal/logger"
	"micro-CRM/internal/middleware"
	"micro-CRM/internal/models"
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
func (a *Api) SetupAllRoutes() {
	// Setup auth router
	a.SetupAuthRouter()

	// Setup Dashboard router
	a.SetupDashRouter()

	// Setup routes
	a.SetupAuthenticationRoutes()
	a.SetupCompanyRoutes()
	a.SetupContactRoutes()
	a.SetupFileRoutes()
	a.SetupTaskRoutes()
	a.SetupHealthRoutes()
	a.SetupInteractionRoutes()
	a.SetupDashboardRoutes()
}
func (a *Api) SetupAuthenticationRoutes() {
	a.router.HandleFunc("/register", a.CRMHandlers.RegisterUser).Methods("POST")
	a.router.HandleFunc("/login", a.CRMHandlers.LoginUser).Methods("POST")
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
	a.authRouter.HandleFunc("/files", a.CRMHandlers.CreateFile).Methods("POST")
	a.authRouter.HandleFunc("/files", a.CRMHandlers.ListFiles).Methods("GET")
	a.authRouter.HandleFunc("/files/upload", a.CRMHandlers.UploadFileHandler).Methods("POST")
	a.authRouter.HandleFunc("/files/{id}", a.CRMHandlers.GetFile).Methods("GET")
	a.authRouter.HandleFunc("/files/{id}", a.CRMHandlers.UpdateFile).Methods("PUT")
	a.authRouter.HandleFunc("/files/{id}", a.CRMHandlers.DeleteFile).Methods("DELETE")
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
}
func (a *Api) SetupLogger() {
	a.log = logger.NewConsoleLogger(os.Stderr, "[CRM-API]", 0, logger.LogLevelInfo)
	a.CRMHandlers.Log = a.log
	a.DBManager.Log = a.log
	a.log.Info("Custom Logger initialized")
}
func (a *Api) SetupHealthRoutes() {
	a.router.HandleFunc("/health/API", a.CRMHandlers.Hello).Methods("GET")
	a.router.HandleFunc("/health/DB", a.CRMHandlers.DBPing).Methods("GET")
}
func (a *Api) Start() {
	var startErr error
	// Setting Up logger
	a.SetupLogger()

	a.log.Info("Starting CRM API")

	// Token setting
	a.log.Info("Setting JWT token")
	utils.SetJWTSecret(a.Params.JWTToken)

	// Database initialization
	a.log.Info("Setting DB connection")
	manager := database.NewDBManager(a.Params.DbPath)
	startErr = manager.Connect()
	if startErr != nil {
		a.log.Fatal("Cannot connect to database", startErr)
	}
	err := manager.ApplyMigrations()
	if err != nil {
		a.log.Fatal("Cannot start Database : ", err)
	}
	a.db = manager.DB

	// Router initialization
	a.router = mux.NewRouter()
	a.CRMHandlers.DB = manager.DB

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
		if err = server.ListenAndServeTLS(a.Params.CertFilePath, a.Params.KeyFilePath); err != nil && !errors.Is(http.ErrServerClosed, err) {
			log.Fatalf("Cannot start API: %v", err)
		}
	}()

	<-killSignal // Block until a signal is received
	log.Println("Received shutdown signal. Initiating graceful shutdown...")

	// Context creation and graceful shutdown of server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 5 seconds to shut down
	defer cancel()
	if err = server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	a.Stop()
}
func (a *Api) Stop() {
	a.log.Info("Graceful shutdown of services")
	err := a.db.Close()
	if err != nil {
		a.log.Warn("Couldn't close database connection:", err)
	}
	a.log.Info("API shutdown gracefully")
}
