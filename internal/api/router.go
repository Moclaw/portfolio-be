package api

import (
	"net/http"
	"portfolio-be/internal/api/handlers"
	"portfolio-be/internal/api/middleware"
	"portfolio-be/internal/config"
	"portfolio-be/internal/repository"
	"portfolio-be/internal/services"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB, s3Service *services.S3Service, cfg *config.Config) *gin.Engine {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Portfolio Backend API is running",
		})
	})

	// Swagger documentation endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Initialize repositories
	contentRepo := repository.NewContentRepository(db)
	uploadRepo := repository.NewUploadRepository(db)
	resourceRepo := repository.NewResourceRepository(db)
	experienceRepo := repository.NewExperienceRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	technologyRepo := repository.NewTechnologyRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	testimonialRepo := repository.NewTestimonialRepository(db)
	userRepo := repository.NewUserRepository(db)
	contactRepo := repository.NewContactRepository(db)

	// Initialize services
	contentService := services.NewContentService(contentRepo)
	uploadService := services.NewUploadService(uploadRepo, s3Service)
	resourceService := services.NewResourceService(resourceRepo, uploadRepo, s3Service)
	experienceService := services.NewExperienceService(experienceRepo)
	serviceService := services.NewServiceService(serviceRepo)
	technologyService := services.NewTechnologyService(technologyRepo)
	projectService := services.NewProjectService(projectRepo)
	testimonialService := services.NewTestimonialService(testimonialRepo)
	jwtService := services.NewJWTService(cfg.JWTConfig.SecretKey, cfg.JWTConfig.Issuer)
	authService := services.NewAuthService(userRepo, jwtService)
	contactService := services.NewContactService(contactRepo)

	// Initialize Cron Service
	cronService := services.NewCronService(resourceService, uploadService)
	// Start cron service in background
	go cronService.Start()

	// Initialize handlers
	contentHandler := handlers.NewContentHandler(contentService)
	uploadHandler := handlers.NewUploadHandler(uploadService)
	resourceHandler := handlers.NewResourceHandler(resourceService)
	experienceHandler := handlers.NewExperienceHandler(experienceService)
	serviceHandler := handlers.NewServiceHandler(serviceService)
	technologyHandler := handlers.NewTechnologyHandler(technologyService)
	projectHandler := handlers.NewProjectHandler(projectService)
	testimonialHandler := handlers.NewTestimonialHandler(testimonialService)
	portfolioHandler := handlers.NewPortfolioHandler(experienceService, serviceService, technologyService, projectService, testimonialService)
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userRepo)
	contactHandler := handlers.NewContactHandler(contactService)
	statsHandler := handlers.NewStatsHandler(projectService, experienceService, technologyService, serviceService, testimonialService, contactService)
	adminOrderHandler := handlers.NewAdminOrderHandler(projectService, experienceService, technologyService, serviceService, testimonialService)

	// Portfolio endpoint (combined data)
	router.GET("/api/portfolio", portfolioHandler.GetPortfolioData)

	// Authentication routes
	auth := router.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/register", authHandler.Register)
		auth.POST("/refresh", authHandler.RefreshToken)

		// Protected auth routes
		authProtected := auth.Group("")
		authProtected.Use(middleware.AuthMiddleware(jwtService))
		{
			authProtected.GET("/profile", authHandler.Profile)
			authProtected.POST("/logout", authHandler.Logout)
		}
	}

	// Admin routes (protected)
	admin := router.Group("/admin")
	admin.Use(middleware.AuthMiddleware(jwtService))
	admin.Use(middleware.AdminMiddleware())
	{
		// User management
		admin.GET("/users", userHandler.GetUsers)
		admin.POST("/users", userHandler.CreateUser)
		admin.GET("/users/:id", userHandler.GetUser)
		admin.PUT("/users/:id", userHandler.UpdateUser)
		admin.PATCH("/users/:id/password", userHandler.UpdateUserPassword)
		admin.DELETE("/users/:id", userHandler.DeleteUser)
		admin.PATCH("/users/:id/toggle-status", userHandler.ToggleUserStatus)

		// Content management
		admin.POST("/contents", contentHandler.CreateContent)
		admin.PUT("/contents/:id", contentHandler.UpdateContent)
		admin.DELETE("/contents/:id", contentHandler.DeleteContent)

		// Experience management
		admin.POST("/experiences", experienceHandler.CreateExperience)
		admin.PUT("/experiences/:id", experienceHandler.UpdateExperience)
		admin.DELETE("/experiences/:id", experienceHandler.DeleteExperience)

		// Service management
		admin.POST("/services", serviceHandler.CreateService)
		admin.PUT("/services/:id", serviceHandler.UpdateService)
		admin.DELETE("/services/:id", serviceHandler.DeleteService)

		// Technology management
		admin.POST("/technologies", technologyHandler.CreateTechnology)
		admin.PUT("/technologies/:id", technologyHandler.UpdateTechnology)
		admin.DELETE("/technologies/:id", technologyHandler.DeleteTechnology)

		// Project management
		admin.POST("/projects", projectHandler.CreateProject)
		admin.PUT("/projects/:id", projectHandler.UpdateProject)
		admin.DELETE("/projects/:id", projectHandler.DeleteProject)

		// Testimonial management
		admin.POST("/testimonials", testimonialHandler.CreateTestimonial)
		admin.PUT("/testimonials/:id", testimonialHandler.UpdateTestimonial)
		admin.DELETE("/testimonials/:id", testimonialHandler.DeleteTestimonial)

		// Contact management
		admin.GET("/contacts", contactHandler.GetContacts)
		admin.GET("/contacts/:id", contactHandler.GetContact)
		admin.PUT("/contacts/:id", contactHandler.UpdateContact)
		admin.DELETE("/contacts/:id", contactHandler.DeleteContact)
		admin.GET("/contacts/unread-count", contactHandler.GetUnreadCount)
		admin.PATCH("/contacts/:id/mark-read", contactHandler.MarkAsRead)

		// Upload management
		admin.POST("/uploads", uploadHandler.UploadFile)
		admin.DELETE("/uploads/:id", uploadHandler.DeleteUpload)

		// Resource management
		admin.POST("/resources", resourceHandler.CreateResource)
		admin.GET("/resources", resourceHandler.GetAllResources)
		admin.GET("/resources/:id", resourceHandler.GetResource)
		admin.PUT("/resources/:id", resourceHandler.UpdateResource)
		admin.DELETE("/resources/:id", resourceHandler.DeleteResource)
		admin.GET("/resources/stats", resourceHandler.GetResourceStats)
		admin.POST("/resources/refresh-urls", resourceHandler.RefreshExpiredURLs)

		// Order management
		admin.PUT("/projects/order", adminOrderHandler.UpdateProjectsOrder)
		admin.PUT("/experiences/order", adminOrderHandler.UpdateExperiencesOrder)
		admin.PUT("/technologies/order", adminOrderHandler.UpdateTechnologiesOrder)
		admin.PUT("/services/order", adminOrderHandler.UpdateServicesOrder)
		admin.PUT("/testimonials/order", adminOrderHandler.UpdateTestimonialsOrder)
	}

	// Public API routes (read-only)
	api := router.Group("/api")
	{
		// Content routes
		api.GET("/contents", contentHandler.GetAllContents)
		api.GET("/contents/:id", contentHandler.GetContent)

		// Experience routes
		api.GET("/experiences", experienceHandler.GetExperiences)
		api.GET("/experiences/:id", experienceHandler.GetExperience)

		// Service routes
		api.GET("/services", serviceHandler.GetAllServices)
		api.GET("/services/:id", serviceHandler.GetService)

		// Technology routes
		api.GET("/technologies", technologyHandler.GetTechnologies)
		api.GET("/technologies/:id", technologyHandler.GetTechnology)

		// Project routes
		api.GET("/projects", projectHandler.GetProjects)
		api.GET("/projects/:id", projectHandler.GetProject)

		// Testimonial routes
		api.GET("/testimonials", testimonialHandler.GetTestimonials)
		api.GET("/testimonials/:id", testimonialHandler.GetTestimonial)

		// Contact routes (for submitting contact forms)
		api.POST("/contacts", contactHandler.CreateContact)

		// Stats routes
		api.GET("/stats/counts", statsHandler.GetCounts)

		// Upload routes
		api.GET("/uploads", uploadHandler.GetAllUploads)
		api.GET("/uploads/:id", uploadHandler.GetUpload)
		api.GET("/uploads/summary", uploadHandler.GetAllUploadsWithSummary)

		// Resource routes (public read-only)
		api.GET("/resources", resourceHandler.GetAllResources)
		api.GET("/resources/:id", resourceHandler.GetResource)
		api.POST("/resources/:id/download", resourceHandler.DownloadResource)
		api.GET("/resources/stats", resourceHandler.GetResourceStats)
	}

	// Legacy API v1 routes (kept for backward compatibility, read-only)
	v1 := router.Group("/api/v1")
	{
		// Content routes
		contents := v1.Group("/contents")
		{
			contents.GET("", contentHandler.GetAllContents)
			contents.GET("/:id", contentHandler.GetContent)
		}

		// Upload routes
		uploads := v1.Group("/uploads")
		{
			uploads.GET("", uploadHandler.GetAllUploads)
			uploads.GET("/:id", uploadHandler.GetUpload)
			uploads.GET("/summary", uploadHandler.GetAllUploadsWithSummary)
		}

		// Resource routes
		resources := v1.Group("/resources")
		{
			resources.GET("", resourceHandler.GetAllResources)
			resources.GET("/:id", resourceHandler.GetResource)
			resources.POST("/:id/download", resourceHandler.DownloadResource)
		}

		// Experience routes
		experiences := v1.Group("/experiences")
		{
			experiences.GET("", experienceHandler.GetExperiences)
			experiences.GET("/:id", experienceHandler.GetExperience)
		}

		// Service routes
		services := v1.Group("/services")
		{
			services.GET("", serviceHandler.GetAllServices)
			services.GET("/:id", serviceHandler.GetService)
		}

		// Technology routes
		technologies := v1.Group("/technologies")
		{
			technologies.GET("", technologyHandler.GetTechnologies)
			technologies.GET("/:id", technologyHandler.GetTechnology)
		}

		// Project routes
		projects := v1.Group("/projects")
		{
			projects.GET("", projectHandler.GetProjects)
			projects.GET("/:id", projectHandler.GetProject)
		}

		// Testimonial routes
		testimonials := v1.Group("/testimonials")
		{
			testimonials.GET("", testimonialHandler.GetTestimonials)
			testimonials.GET("/:id", testimonialHandler.GetTestimonial)
		}

		// Stats routes
		stats := v1.Group("/stats")
		{
			stats.GET("/counts", statsHandler.GetCounts)
		}
	}

	return router
}
