# Next Steps — Path Authentication System Testing & Validation

**Date Created:** 2026-06-05  
**Current Status:** v0.4.0 Deployed, Ready for Testing  
**Priority:** HIGH — Complete user-facing testing before full rollout

---

## Immediate Actions (Next 15 minutes)

### 1. Restart Coordinator with Latest Binary ⚠️
**Status:** PENDING  
**Action:**
```powershell
# Make sure old process is stopped
Get-Process arcvault-coordinator -ErrorAction SilentlyContinue | Stop-Process -Force

# Start with latest binary (includes updated dashboard)
cd C:\ArcVault
$env:ARCVAULT_CREDENTIAL_KEY = "ec53c0dd0844aae1f216f357ccd79c3e2c7417281fc04b44662026ce88988b14"
.\arcvault-coordinator.exe start
```
**Expected:** Coordinator listening on port 8080

### 2. Verify Credentials in Navigation
**Status:** PENDING  
**Location:** http://localhost:8080  
**Check:**
- [ ] Refresh browser (Ctrl+Shift+R for hard refresh)
- [ ] Login with admin token: `prod-5b115573-f8d8-4a11-ab4a-0873fb95aa0`
- [ ] Look for **"Credentials"** in main navigation (between Jobs and History)
- [ ] Click on it
- [ ] Should see "+ New Credential" button at top right

**If NOT visible:**
1. Check browser console (F12) for JavaScript errors
2. Try navigating directly to: `http://localhost:8080/#/admin/credentials`
3. Check if dashboard CSS loaded (look for styled header)

---

## Phase 1: Dashboard UI Testing (30 minutes)

### 3. Test Credential Creation Form
**Status:** PENDING  
**Credentials Page:** http://localhost:8080/#/admin/credentials

#### Test Case: SMB Credential
- [ ] Click "+ New Credential"
- [ ] Enter Name: "Test-SMB-Share"
- [ ] Select Type: "SMB (Windows File Share)"
- [ ] Fill fields:
  - Host: `\\server\share`
  - Username: `domain\user`
  - Password: `testpass123`
- [ ] Click "Create Credential"
- [ ] Verify: Credential appears in list below
- [ ] Check: No plaintext password visible in list (encrypted at rest)

#### Test Case: SSH Credential
- [ ] Click "+ New Credential"
- [ ] Enter Name: "Test-SSH-Key"
- [ ] Select Type: "SSH (Linux/Unix)"
- [ ] Select Auth Type: "Private Key"
- [ ] Paste valid SSH private key (PEM format)
- [ ] Click "Create Credential"
- [ ] Verify: Credential appears in list

#### Test Case: Database Credential
- [ ] Click "+ New Credential"
- [ ] Enter Name: "Test-DB"
- [ ] Select Type: "Database"
- [ ] Fill fields:
  - Host: `localhost`
  - Port: `5432`
  - Username: `postgres`
  - Password: `testpass123`
- [ ] Click "Create Credential"
- [ ] Verify: Credential appears in list

### 4. Test Credential List Security
**Status:** PENDING
- [ ] Open browser DevTools (F12)
- [ ] Go to Network tab
- [ ] Click on GET request to `/api/credential-profiles`
- [ ] Check Response tab
- [ ] Verify: No `encrypted_data` field visible
- [ ] Verify: Only id, name, type, created_at returned

### 5. Test Credential Deletion
**Status:** PENDING
- [ ] Click "Delete" button on a credential
- [ ] Confirm deletion
- [ ] Verify: Credential removed from list
- [ ] Verify: Success toast/message shown

---

## Phase 2: Job Integration Testing (45 minutes)

### 6. Test Credential Assignment to Job
**Status:** PENDING  
**Location:** http://localhost:8080/#/jobs

Steps:
- [ ] Click "+ New Job"
- [ ] Fill basic fields:
  - Name: "Test-Job-With-Credentials"
  - Source: `/tmp/source`
  - Destination: `/tmp/dest`
- [ ] Select an Agent (if available)
- [ ] Look for "Path Authentication" dropdown
- [ ] Verify: List shows only compatible credentials for that agent OS
  - Windows agent → SMB credentials only
  - Linux agent → SSH credentials only
- [ ] Select a credential from dropdown
- [ ] Create job
- [ ] Verify: Job created with credential attached

### 7. Test Credential Display in Job Details
**Status:** PENDING
- [ ] Click on the job you just created
- [ ] Verify: Shows which credential is assigned
- [ ] Verify: Credential name visible, but not the actual secrets

### 8. Test 409 Conflict Error
**Status:** PENDING
- [ ] Go back to Credentials page
- [ ] Try to delete a credential that's assigned to a job
- [ ] Expected: 409 Conflict error message
- [ ] Error message should say: "Cannot delete — it's referenced by one or more jobs"

---

## Phase 3: Agent Integration Testing (if agents available) (60 minutes)

### 9. Agent Credential Reception
**Status:** PENDING  
**Prerequisites:** At least one agent connected to coordinator

Steps:
- [ ] Create a job with credentials
- [ ] Assign to an available agent
- [ ] Monitor agent logs (or coordinator logs)
- [ ] Verify agent receives job with credentials
- [ ] Check: Credentials are decrypted before sending to agent
- [ ] Check: Credentials NOT visible in coordinator logs

### 10. Test Credential Application (Windows SMB)
**Status:** PENDING
- [ ] Create SMB credential with real Windows share details
- [ ] Create job assigned to this credential
- [ ] Run job on Windows agent
- [ ] Agent should execute: `cmdkey /add:host /user:user /pass:pass`
- [ ] Verify: Agent can access SMB share with injected credentials
- [ ] Verify: Credentials removed after job completes

### 11. Test Credential Application (Linux SSH)
**Status:** PENDING
- [ ] Create SSH credential with real key
- [ ] Create job assigned to this credential
- [ ] Run job on Linux agent
- [ ] Agent should:
  - Create temp file with SSH key
  - Set SSH_KEY_PATH environment variable
  - Execute job
  - Clean up temp file
- [ ] Verify: Job executes successfully with SSH access

---

## Phase 4: Error Handling & Edge Cases (30 minutes)

### 12. Test Missing Encryption Key
**Status:** PENDING
- [ ] Stop coordinator
- [ ] Restart without setting `ARCVAULT_CREDENTIAL_KEY` env var
- [ ] Try to create credential via API or UI
- [ ] Expected: 503 Service Unavailable error
- [ ] Error message: "Encryption key not configured"

### 13. Test Invalid Credential Type
**Status:** PENDING
- [ ] Try to assign SMB credential to Linux agent (via API if possible)
- [ ] Expected: 422 Unprocessable Entity
- [ ] Error message: "Credential type SMB incompatible with agent OS"

### 14. Test Credential Data Validation
**Status:** PENDING
- [ ] Try to create SMB credential with missing Host field
- [ ] Expected: Validation error before submission
- [ ] Try via API with incomplete data
- [ ] Expected: 400 Bad Request

### 15. Test Concurrent Credential Operations
**Status:** PENDING
- [ ] Create multiple credentials in quick succession
- [ ] Delete while creating others
- [ ] Verify: No data corruption
- [ ] Verify: All operations complete successfully

---

## Phase 5: Database & Audit Trail (20 minutes)

### 16. Verify Database Schema
**Status:** PENDING
- [ ] Check SQLite database structure
- [ ] Verify `credential_profiles` table exists with columns:
  - id (TEXT PRIMARY KEY)
  - name (TEXT UNIQUE)
  - type (TEXT)
  - encrypted_data (BLOB)
  - created_at (TEXT)

### 17. Verify Job Snapshots
**Status:** PENDING
- [ ] Run a job with credentials
- [ ] Check `job_runs` table
- [ ] Verify columns exist:
  - credential_profile_id
  - credential_profile_name
- [ ] Verify: Values populated from execution

### 18. Test Audit Trail
**Status:** PENDING
- [ ] Delete credential that was used by past job
- [ ] Check `job_runs` record
- [ ] Verify: Still shows credential_profile_id and name (not lost)
- [ ] This proves audit trail persists even if credential deleted

---

## Phase 6: Performance & Load (Optional, if time)

### 19. Test with Many Credentials
**Status:** OPTIONAL
- [ ] Create 100+ credentials
- [ ] Verify: List page still loads quickly
- [ ] Verify: No UI lag
- [ ] Verify: Filtering by type still works

### 20. Test Credential Reuse
**Status:** OPTIONAL
- [ ] Create 10 jobs, all referencing same credential
- [ ] Verify: Encryption key not re-derived each time
- [ ] Performance: Should be O(1) per job

---

## Documentation Tasks

### 21. Update User Documentation
**Status:** PENDING
- [ ] Create User Guide for Credentials feature
- [ ] Document credential types and required fields
- [ ] Document job-credential workflow
- [ ] Add screenshots of UI

### 22. Update API Documentation
**Status:** PENDING
- [ ] Document `/api/credential-profiles` endpoints
- [ ] Add request/response examples
- [ ] Document error codes and meanings

### 23. Create Troubleshooting Guide
**Status:** PENDING
- [ ] Common issues and solutions
- [ ] How to verify credentials are encrypted
- [ ] How to rotate encryption key
- [ ] How to backup/restore credentials

---

## Production Readiness Checklist

### Security
- [ ] Encryption key backed up securely (not in git)
- [ ] Admin token backed up securely (not in git)
- [ ] No credentials logged in plaintext
- [ ] No credentials exposed in API responses
- [ ] Credential deletion respects FK constraints (409 errors)

### Functionality
- [ ] Credentials create/read/update/delete working
- [ ] Type validation working (incompatible OS combinations blocked)
- [ ] Agent receives credentials in job response
- [ ] Agent applies credentials correctly
- [ ] Credentials cleaned up after job execution

### Operations
- [ ] Database backups working
- [ ] Service can start/stop cleanly
- [ ] Coordinator logs audit trail
- [ ] No data corruption on unexpected shutdown
- [ ] Key rotation procedure documented and tested

### Performance
- [ ] Credential list loads < 500ms
- [ ] Credential creation < 1s
- [ ] No memory leaks on repeated operations
- [ ] Encryption/decryption acceptable performance

---

## Known Limitations to Address

### 1. Dashboard Styling
- Credentials nav link recently added
- May need CSS adjustments for consistency
- Check hover states, active states

### 2. Credential Type Restrictions
- Currently enforced at API level
- Could add UI hints (e.g., "Windows only")
- Consider allowing future "cross-platform" type

### 3. No Credential Expiration
- Current implementation: credentials never expire
- Future: Add expiry field and warning system

### 4. No Credential Versioning
- Can't rotate credential value without deleting
- Breaks jobs referencing it
- Future: Version credentials, migrate references

### 5. Limited Credential Types
- Current: SMB, SSH, AWS, Database
- Future: Azure, GCP, Kubernetes, custom types

---

## Bug Tracking

### To Report
- [ ] Button visibility in Credentials page
- [ ] Form field validation messages
- [ ] API error message clarity
- [ ] Performance issues with many credentials

### To Investigate
- [ ] Why one staging test didn't show output (7/8)
- [ ] Potential race conditions in concurrent ops
- [ ] Database locking on high concurrency

---

## Roll-Out Plan

### Staging Validation (This Session)
- Complete all Phase 1-2 testing
- Fix any critical UI issues
- Verify database integrity

### Beta Testing (Next Session)
- Enable for subset of users
- Collect feedback on UX
- Monitor logs for issues
- Performance testing

### Full Production (When Ready)
- All tests passing
- Documentation complete
- Team trained
- Rollback plan verified
- Monitoring/alerting in place

---

## Success Criteria

✅ **Session Complete When:**
1. Credentials button visible in navigation
2. Can create all credential types
3. Credentials don't expose encrypted data
4. Can assign to jobs
5. Type validation working
6. Delete conflict handling working
7. No critical UI issues

⏱️ **Estimated Time:** 2-3 hours for complete testing

---

## Quick Reference

### Credentials Page
- **URL:** `http://localhost:8080/#/admin/credentials`
- **Component:** `dashboard/src/views/admin/Credentials.vue`
- **Route:** `/admin/credentials`

### API Endpoint
- **POST** `/api/credential-profiles` — Create credential
- **GET** `/api/credential-profiles` — List credentials  
- **DELETE** `/api/credential-profiles/{id}` — Delete credential

### Production Credentials
- **Admin Token:** `prod-5b115573-f8d8-4a11-ab4a-0873fb95aa0`
- **Encryption Key:** `ec53c0dd0844aae1f216f357ccd79c3e2c7417281fc04b44662026ce88988b14`

### Coordinator Status
- **Binary:** `C:\ArcVault\arcvault-coordinator.exe` (v0.4.0)
- **Port:** 8080
- **Database:** `C:\ArcVault\arcvault.db`

---

**Last Updated:** 2026-06-05 14:00 PM  
**Next Session:** Test Credentials UI and complete integration validation
