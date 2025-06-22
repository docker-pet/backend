package models

type OutlineServerSyncType string

const (
	OutlineRemoteSync OutlineServerSyncType = "remote"
	OutlineLocalSync  OutlineServerSyncType = "local"
)
