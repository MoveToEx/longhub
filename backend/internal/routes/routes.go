package routes

import (
	"long/internal/handlers"
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
		public.GET("/image/:id", handlers.GetImage)
		public.GET("/image/:id/version", handlers.GetImageVersions)
		public.GET("/user/:id/contribution", handlers.GetUserContribution)
		public.GET("/user/:id", handlers.GetUser)
		public.GET("/tag/autocomplete", handlers.TagAutocomplete)
	}

	protected := r.Group("/")
	protected.Use(middleware.Auth(true))
	{
		protected.GET("/auth", handlers.GetIdentity)
		protected.POST("/image/sign", handlers.CreateUploadSession)
		protected.POST("/image/ack", handlers.AcknowledgeSession)
		protected.PATCH("/image/:id", handlers.UpdateImage)
		protected.GET("/user", handlers.GetSelf)
		protected.PATCH("/user/preferences", handlers.UpdatePreference)

		// Favorite
		protected.GET("/favorite", handlers.GetFavorites)
		protected.GET("/favorite/:id", handlers.GetFavoriteState)
		protected.PATCH("/favorite/:id", handlers.SetFavoriteShortcut)
		protected.DELETE("/favorite/:id", handlers.DeleteFavorite)
		protected.POST("/favorite", handlers.AddFavorite)

		protected.GET("/recommend", handlers.GetRecommendedTags)

		// WebAuthn
		protected.GET("/user/webauthn", handlers.GetPasskey)
		protected.POST("/user/webauthn/new", handlers.BeginAddPasskey)
		protected.POST("/user/webauthn/validate", handlers.ValidateAddPasskey)
		protected.PATCH("/user/webauthn/:id", handlers.EditPasskey)
		protected.DELETE("/user/webauthn/:id", handlers.DeletePasskey)

		// Integration
		protected.POST("/user/appkey", handlers.CreateAppKey)
		protected.PATCH("/user/appkey/:id", handlers.EditAppKey)
		protected.DELETE("/user/appkey/:id", handlers.DeleteAppKey)
		protected.GET("/user/appkey", handlers.GetAppKey)
	}

	optional := r.Group("/")
	optional.Use(middleware.Auth(false))
	{
		optional.POST("/image/quick-search", handlers.QuickSearchImages)
		optional.GET("/image", handlers.ListImages)
		protected.GET("/tag/random/:name", handlers.GetRandomImagesByTag)
		optional.GET("/user/:id/image", handlers.ListImagesByUser)
	}

}
