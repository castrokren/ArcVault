package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"

	"arcvault/coordinator/internal/credcrypto"
)

// CredentialProfile matches the database model but excludes encrypted data in JSON
type CredentialProfileResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
}

// handleCreateCredentialProfile handles POST /api/credential-profiles
func (s *Server) handleCreateCredentialProfile(w http.ResponseWriter, r *http.Request) {
	// Check if encryption key is set
	if os.Getenv("ARCVAULT_CREDENTIAL_KEY") == "" {
		http.Error(w, "encryption key not configured", http.StatusServiceUnavailable)
		return
	}

	// Load encryption key
	key, err := credcrypto.LoadKey()
	if err != nil {
		http.Error(w, "encryption key error", http.StatusServiceUnavailable)
		return
	}

	var input struct {
		Name string                 `json:"name"`
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if input.Name == "" || input.Type == "" || input.Data == nil {
		http.Error(w, "name, type, and data are required", http.StatusBadRequest)
		return
	}

	// Serialize and encrypt the data
	dataJSON, err := json.Marshal(input.Data)
	if err != nil {
		http.Error(w, "failed to serialize data", http.StatusBadRequest)
		return
	}

	encryptedData, err := credcrypto.Encrypt(key, dataJSON)
	if err != nil {
		http.Error(w, "encryption failed", http.StatusInternalServerError)
		return
	}

	// Generate ID and create profile
	id := "cred-" + randomHex(8)

	if err := s.db.CreateCredentialProfile(id, input.Name, input.Type, encryptedData); err != nil {
		http.Error(w, "failed to create credential profile", http.StatusInternalServerError)
		return
	}

	// Retrieve the created profile to get the timestamp
	profile, err := s.db.GetCredentialProfile(id)
	if err != nil {
		http.Error(w, "failed to retrieve created profile", http.StatusInternalServerError)
		return
	}

	response := CredentialProfileResponse{
		ID:        profile.ID,
		Name:      profile.Name,
		Type:      profile.Type,
		CreatedAt: profile.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleListCredentialProfiles handles GET /api/credential-profiles
func (s *Server) handleListCredentialProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := s.db.ListCredentialProfiles()
	if err != nil {
		http.Error(w, "failed to list credential profiles", http.StatusInternalServerError)
		return
	}

	// Convert to response format without encrypted data
	var responses []*CredentialProfileResponse
	for _, p := range profiles {
		responses = append(responses, &CredentialProfileResponse{
			ID:        p.ID,
			Name:      p.Name,
			Type:      p.Type,
			CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// handleDeleteCredentialProfile handles DELETE /api/credential-profiles/{id}
func (s *Server) handleDeleteCredentialProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Check if any jobs reference this profile
	hasRefs, err := s.db.HasJobsReferencingProfile(id)
	if err != nil {
		http.Error(w, "failed to check job references", http.StatusInternalServerError)
		return
	}

	if hasRefs {
		http.Error(w, "credential profile is referenced by one or more jobs", http.StatusConflict)
		return
	}

	// Delete the profile
	err = s.db.DeleteCredentialProfile(id)
	if err != nil {
		http.Error(w, "credential profile not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func randomHex(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)
}
