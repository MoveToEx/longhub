package routes

import (
	"long/internal/handlers"
	userHandlers "long/internal/handlers/user"
	"long/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	public := r.Group("/")
	{
		public.POST("/auth/login", handlers.LoginRoute)
		public.POST("/auth/login/webauthn/new", handlers.BeginWebAuthnLogin)
		public.POST("/auth/login/webauthn/validate", handlers.ValidateWebAuthnLogin)
		public.POST("/auth/register", handlers.RegisterRoute)
		public.GET("/image", handlers.ListImages)
		public.GET("/image/:id", handlers.GetImage)
		public.POST("/image/search", handlers.SearchImages)
		public.GET("/image/:id/version", handlers.GetImageVersions)
		public.GET("/user/:id/contribution", userHandlers.GetUserContribution)
		public.GET("/user/:id", userHandlers.GetUser)
		public.GET("/user/:id/image", userHandlers.ListImagesByUser)

		public.GET("/tag/autocomplete",
			middleware.Header("Cache-Control", "public, max-age=180, s-age=3600"),
			handlers.TagAutocomplete,
		)
	}

	protected := r.Group("/")
	protected.Use(middleware.Auth(true))
	{
		protected.GET("/auth", handlers.GetIdentity)
		protected.POST("/image/sign", handlers.CreateUploadSession)
		protected.POST("/image/ack", handlers.AcknowledgeSession)
		protected.PATCH("/image/:id", handlers.UpdateImage)
		protected.GET("/user", userHandlers.GetSelf)

		// Favorite
		protected.GET("/favorite", handlers.GetFavorites)
		protected.GET("/favorite/:id", handlers.GetFavoriteState)
		protected.PATCH("/favorite/:id", handlers.SetFavoriteShortcut)
		protected.DELETE("/favorite/:id", handlers.DeleteFavorite)
		protected.POST("/favorite", handlers.AddFavorite)

		protected.GET("/recommend", handlers.GetRecommendedTags)

		// WebAuthn
		protected.GET("/user/webauthn", userHandlers.GetPasskey)
		protected.POST("/user/webauthn/new", userHandlers.BeginAddPasskey)
		protected.POST("/user/webauthn/validate", userHandlers.ValidateAddPasskey)
		protected.PATCH("/user/webauthn/:id", userHandlers.EditPasskey)
		protected.DELETE("/user/webauthn/:id", userHandlers.DeletePasskey)

		// Integration
		protected.POST("/user/appkey", userHandlers.CreateAppKey)
		protected.PATCH("/user/appkey/:id", userHandlers.EditAppKey)
		protected.DELETE("/user/appkey/:id", userHandlers.DeleteAppKey)
		protected.GET("/user/appkey", userHandlers.GetAppKey)
		protected.GET("/user/webhook", userHandlers.ListWebhooks)
		protected.POST("/user/webhook", userHandlers.CreateWebhook)
		protected.PATCH("/user/webhook/:id", userHandlers.EditWebhook)
		protected.DELETE("/user/webhook/:id", userHandlers.DeleteWebhook)
	}

	optional := r.Group("/")
	optional.Use(middleware.Auth(false))
	{
		optional.POST("/image/quick-search", handlers.QuickSearchImages)
	}
}
