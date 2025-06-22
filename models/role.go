package models

type UserRole string

const (
    RoleUser  UserRole = "user"
    RoleAdmin UserRole = "admin"
    RoleGuest UserRole = "guest"
)
