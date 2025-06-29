package main

import (
	"net/http"
	"fmt"
	"io"
	"mime"
	"os"
	"crypto/rand"
	"encoding/hex"
	
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
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

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get video from db", err)
		return 
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not permitted", err)
		return
	}
	

	fmt.Println("uploading video", videoID, "by user", userID)

	const uploadLimit = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, uploadLimit)
	defer r.Body.Close()

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return 
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get media Type from header", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Video isn't mp4", err)
		return
	}

	temp, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create temp file", err)
		return
	}
	defer os.Remove(temp.Name())
	defer temp.Close()

	_, err = io.Copy(temp, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy file", err)
		return
	}

	_, err = temp.Seek(0, io.SeekStart
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could't change temp pointer", err)
		return
	}
	
	bytes := make([]byte, 16)
	_, err = rand.Read(bytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error reading random bytes", err)
		return
	}
	
	hexString := hex.EncodeToString(bytes)
	fileKey := hexString + ".mp4"

	_, err = cfg.s3Client.PutObject(context.TODO(), s3.PutObjectInput{
		Bucket:      aws.String("tubely-77247"),
		Key:         aws.String(hexString),
		Body:        temp,
		ContentType: mediaType,
	})	
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to upload file", err)
		return 
	}
	

}
