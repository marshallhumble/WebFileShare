package main

type contextKey string

const isAuthenticatedContextKey = contextKey("isAuthenticated")
const isAdminContextKey = contextKey("isAdmin")
const isGuestContextKey = contextKey("isGuest")
