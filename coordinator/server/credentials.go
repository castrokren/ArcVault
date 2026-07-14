package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"arcvault/coordinator/internal/credcrypto"
)

// CredentialProfile matches the database model but excludes encrypted data in JSON
type CredentialProfileResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
}

// loadCredentialKey returns the encryption key from config or falls back to env var.
func (s *Server) loadCredentialKey() ([]byte, error) {
	if s.cfg.CredentialKey != "" {
		return credcrypto.LoadKeyFromString(s.cfg.CredentialKey)
	}
	return credcrypto.LoadKey()
}

// handleCreateCredentialProfile handles POST /api/credential-profiles
func (s *Server) handleCreateCredentialProfile(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)

	// Load encryption key (config takes priority over env var)
	key, err := s.loadCredentialKey()
	if err != nil {
		http.Error(w, "encryption key not configured", http.StatusServiceUnavailable)
		s.logAudit(r, claims, "credential.create", false, nil, nil)
		return
	}

	var input struct {
		Name string                 `json:"name"`
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		s.logAudit(r, claims, "credential.create", false, nil, nil)
		return
	}

	// Validate required fields
	if input.Name == "" || input.Type == "" || input.Data == nil {
		http.Error(w, "name, type, and data are required", http.StatusBadRequest)
		s.logAudit(r, claims, "credential.create", false, nil, nil)
		return
	}

	// Serialize and encrypt the data
	dataJSON, err := json.Marshal(input.Data)
	if err != nil {
		http.Error(w, "failed to serialize data", http.StatusBadRequest)
		s.logAudit(r, claims, "credential.create", false, nil, nil)
		return
	}

	encryptedData, err := credcrypto.Encrypt(key, dataJSON)
	if err != nil {
		http.Error(w, "encryption failed", http.StatusInternalServerError)
		s.logAudit(r, claims, "credential.create", false, nil, nil)
		return
	}

	// Generate ID and create profile
	id := "cred-" + randomHex(8)

	if err := s.db.CreateCredentialProfile(id, input.Name, input.Type, encryptedData); err != nil {
		http.Error(w, "failed to create credential profile", http.StatusInternalServerError)
		s.logAudit(r, claims, "credential.create", false, nil, nil)
		return
	}

	// Retrieve the created profile to get the timestamp
	profile, err := s.db.GetCredentialProfile(id)
	if err != nil {
		http.Error(w, "failed to retrieve created profile", http.StatusInternalServerError)
		s.logAudit(r, claims, "credential.create", false, strPtr("credential"), strPtr(id))
		return
	}

	response := CredentialProfileResponse{
		ID:        profile.ID,
		Name:      profile.Name,
		Type:      profile.Type,
		CreatedAt: profile.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	s.logAudit(r, claims, "credential.create", true, strPtr("credential"), strPtr(id))

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
	responses := make([]*CredentialProfileResponse, 0)
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
	claims := GetUserClaims(r)
	id := r.PathValue("id")

	// Check if any jobs reference this profile
	hasRefs, err := s.db.HasJobsReferencingProfile(id)
	if err != nil {
		http.Error(w, "failed to check job references", http.StatusInternalServerError)
		s.logAudit(r, claims, "credential.delete", false, strPtr("credential"), strPtr(id))
		return
	}

	if hasRefs {
		http.Error(w, "credential profile is referenced by one or more jobs", http.StatusConflict)
		s.logAudit(r, claims, "credential.delete", false, strPtr("credential"), strPtr(id))
		return
	}

	// Delete the profile
	err = s.db.DeleteCredentialProfile(id)
	if err != nil {
		http.Error(w, "credential profile not found", http.StatusNotFound)
		s.logAudit(r, claims, "credential.delete", false, strPtr("credential"), strPtr(id))
		return
	}

	s.logAudit(r, claims, "credential.delete", true, strPtr("credential"), strPtr(id))

	w.WriteHeader(http.StatusNoContent)
}

func randomHex(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// validateCredentialTypeForAgent checks if credential type is compatible with agent OS
func (s *Server) validateCredentialTypeForAgent(credType, agentOS string) bool {
	// Map credential types to compatible OS values
	osCompatibility := map[string]map[string]bool{
		"SMB":      {"windows": true},
		"SSH":      {"linux": true, "darwin": true, "unix": true},
		"AWS":      {"windows": true, "linux": true, "darwin": true},
		"Database": {"windows": true, "linux": true, "darwin": true},
	}

	if compatible, exists := osCompatibility[credType]; exists {
		return compatible[agentOS]
	}

	// If credential type not recognized, allow it (fail open)
	return true
}

// decryptCredentials decrypts a credential profile's encrypted data.
// A non-nil error means the caller must NOT proceed without credentials: a job
// that binds a profile whose key is unavailable or whose ciphertext won't
// decrypt should fail loudly, not silently run a backup with no credentials.
func (s *Server) decryptCredentials(profileID string) (map[string]interface{}, error) {
	key, err := s.loadCredentialKey()
	if err != nil {
		log.Printf("[credentials] key unavailable for profile %s: %v", profileID, err)
		return nil, err
	}

	profile, err := s.db.GetCredentialProfile(profileID)
	if err != nil {
		log.Printf("[credentials] failed to load profile %s: %v", profileID, err)
		return nil, err
	}
	if profile == nil {
		log.Printf("[credentials] profile %s not found", profileID)
		return nil, fmt.Errorf("credential profile %s not found", profileID)
	}

	plaintext, err := credcrypto.Decrypt(key, profile.EncryptedData)
	if err != nil {
		log.Printf("[credentials] decrypt failed for profile %s: %v", profileID, err)
		return nil, err
	}

	var credentials map[string]interface{}
	if err := json.Unmarshal(plaintext, &credentials); err != nil {
		log.Printf("[credentials] malformed plaintext for profile %s: %v", profileID, err)
		return nil, err
	}

	return credentials, nil
}
