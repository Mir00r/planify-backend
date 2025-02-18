package routes

import (
	"github.com/Mir00r/api-gateway/src/api-gateway/src/api/handlers"
	"github.com/Mir00r/api-gateway/src/api-gateway/src/config"
	_ "github.com/Mir00r/api-gateway/src/api-gateway/src/middlewares/auth"
	"github.com/Mir00r/api-gateway/src/api-gateway/src/middlewares/logging"
	"github.com/Mir00r/api-gateway/src/api-gateway/src/middlewares/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Router handles all routing logic for the API Gateway
type Router struct {
	config   *config.Config
	engine   *gin.Engine
	handlers *handlers.Handlers
}

// NewRouter creates a new router instance
func NewRouter(cfg *config.Config, handlers *handlers.Handlers) *Router {
	return &Router{
		config:   cfg,
		engine:   gin.New(),
		handlers: handlers,
	}
}

// Setup configures all routes and middleware
func (r *Router) Setup() {
	// Use custom recovery middleware
	r.engine.Use(logging.Recovery())

	// Setup global middleware
	r.engine.Use(logging.RequestLogger())
	r.engine.Use(ratelimit.GlobalRateLimit(r.config))

	// Health check endpoint
	r.engine.GET("/health", r.handlers.HealthCheck)

	// Metrics endpoint
	r.engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	{
		// Public routes
		public := v1.Group("/public")
		{
			public.POST("/login", r.handlers.Auth.Login)
			public.POST("/register", r.handlers.Auth.Register)
		}

		// Protected routes
		//protected := v1.Group("/")
		//protected.Use(auth.JWTAuth(r.config))
		//{
		//	// User service routes
		//	users := protected.Group("/users")
		//	{
		//		users.GET("/", r.handlers.Users.GetUsers)
		//		users.GET("/:id", r.handlers.Users.GetUser)
		//		users.PUT("/:id", r.handlers.Users.UpdateUser)
		//		users.DELETE("/:id", r.handlers.Users.DeleteUser)
		//	}
		//
		//	// Notification service routes
		//	notifications := protected.Group("/notifications")
		//	{
		//		notifications.GET("/", r.handlers.Notifications.GetNotifications)
		//		notifications.POST("/", r.handlers.Notifications.CreateNotification)
		//	}
		//
		//	// Appointment service routes
		//	appointments := protected.Group("/appointments")
		//	{
		//		appointments.GET("/", r.handlers.Appointments.GetAppointments)
		//		appointments.POST("/", r.handlers.Appointments.CreateAppointment)
		//		appointments.PUT("/:id", r.handlers.Appointments.UpdateAppointment)
		//		appointments.DELETE("/:id", r.handlers.Appointments.DeleteAppointment)
		//	}
		//}
	}

	// Documentation routes
	r.engine.GET("/docs/*any", r.handlers.ServeDocs)
}

// GetEngine returns the configured Gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}
