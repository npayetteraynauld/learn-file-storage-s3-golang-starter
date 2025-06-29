package main

import (
	"fmt"
	"net/http"
	"io"
	"path/filepath"
	"os"
	"mime"
	"crypto/rand"
	"encoding/base64"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}


	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
	return 
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")

	/*dat, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to read image file", err)
		return
	}*/

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get video", err)
		return 
	}

	if video.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}	

	mediaType, _, _ := mime.ParseMediaType(contentType)
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Not a valid image format", nil)
		return 
	}
	
	exts, _ := mime.ExtensionsByType(contentType)
	ext := ".bin"
	if len(exts) > 0 {
		ext = exts[0] 
	}

	byteSlice := make([]byte, 32)
	_, err = rand.Read(byteSlice)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't generate random", err)
		return
	}
	encoded := base64.RawURLEncoding.EncodeToString(byteSlice)

	fileName := fmt.Sprintf("%v%v", encoded, ext)
	videoFilePath := filepath.Join(cfg.assetsRoot, fileName)

	newFile, err := os.Create(videoFilePath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't create file", err)
		return
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "couldn't copy file", err)
		return 
	}

	newURL := fmt.Sprintf("http://localhost:%v/assets/%v", cfg.port, fileName)
	video.ThumbnailURL = &newURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't update video", err)
		return 
	}

	respondWithJSON(w, http.StatusOK, database.Video{
		ID: video.ID,
		CreatedAt: video.CreatedAt,
		UpdatedAt: video.UpdatedAt,
		ThumbnailURL: video.ThumbnailURL,
		VideoURL: video.VideoURL,
		CreateVideoParams: database.CreateVideoParams{
			Title: video.CreateVideoParams.Title,
			Description: video.CreateVideoParams.Description,
			UserID: video.CreateVideoParams.UserID,
		},
	})
}
