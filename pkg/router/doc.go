// Package router provides a custom HTTP router with pattern matching,
// middleware support, and URL parameter extraction.
//
// The router supports the following patterns:
//   - Exact match: /users/list
//   - Named parameters: /users/:id
//   - Wildcard matching: /api/v1/*
//   - Nested parameters: /posts/:id/comments/:commentId
//
// Example usage:
//
//	r := router.New()
//	r.GET("/users/:id", UserHandler)
//	r.GET("/posts/:id/comments/:commentId", CommentHandler)
//	r.GET("/api/v1/*", APIHandler)
//	http.ListenAndServe(":8080", r)
package router
