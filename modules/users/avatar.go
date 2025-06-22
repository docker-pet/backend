package users

import (
	"crypto/md5"
	"fmt"
	"mime"

	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func (m *UsersModule) UploadAvatar(user *models.User, url string) bool {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(url)))
	if user.AvatarHash() == hash {
		return false
	}

	// Download the avatar image from the provided URL
	response, err := m.Ctx.HttpClient.R().Get(url)
	if err != nil {
		m.Logger.Warn(
			"Failed to download avatar",
			"UserId", user.Id,
			"AvatarUrl", url,
			"Error", err,
		)
		return false
	}

	// Detect extension
	contentType := response.Header().Get("Content-Type")
	extensions, err := mime.ExtensionsByType(contentType)
	extension := ".jpg"
	if err == nil && len(extensions) > 0 {
		extension = extensions[0]
	}

	// Filename
	filename := fmt.Sprintf("avatar_%s%s", user.Id, extension)

	// Create a new file from the downloaded bytes
	file, err := filesystem.NewFileFromBytes(response.Bytes(), filename)
	if err != nil {
		m.Logger.Warn(
			"Failed to create avatar file from bytes",
			"UserId", user.Id,
			"Error", err,
		)
		return false
	}

	// Set the avatar for the user
	user.SetAvatar(file, hash)
	return true
}
