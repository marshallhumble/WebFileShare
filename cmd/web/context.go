package main

type contextKey string

const isAuthenticatedContextKey = contextKey("isAuthenticated")
const isAdminContextKey = contextKey("isAdmin")
const isUserContextKey = contextKey("isUser")
const isGuestContextKey = contextKey("isGuest")
